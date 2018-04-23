package controller

import (
	"fmt"
	"time"

	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/util/errors"

	rapi "github.com/amadeusitgroup/redis-operator/pkg/api/redis/v1"
	"github.com/amadeusitgroup/redis-operator/pkg/controller/clustering"
	"github.com/amadeusitgroup/redis-operator/pkg/controller/sanitycheck"
	"github.com/amadeusitgroup/redis-operator/pkg/redis"
)

func (c *Controller) clusterAction(admin redis.AdminInterface, cluster *rapi.RedisCluster, infos *redis.ClusterInfos) (bool, error) {
	var err error
	// Start more pods in needed
	if needMorePods(cluster) {
		if setScalingCondition(&cluster.Status, true) {
			if cluster, err = c.updateHandler(cluster); err != nil {
				return false, err
			}
		}
		pod, err2 := c.podControl.CreatePod(cluster)
		if err2 != nil {
			glog.Errorf("[clusterAction] unable to create a pod associated to the RedisCluster: %s/%s, err: %v", cluster.Namespace, cluster.Name, err2)
			return false, err2
		}

		glog.V(3).Infof("[clusterAction]create a Pod %s/%s", pod.Namespace, pod.Name)
		return true, nil
	}
	if setScalingCondition(&cluster.Status, false) {
		if cluster, err = c.updateHandler(cluster); err != nil {
			return false, err
		}
	}

	// Reconfigure the Cluster if needed
	hasChanged, err := c.applyConfiguration(admin, cluster)
	if err != nil {
		glog.Errorf("[clusterAction] cluster %s/%s, an error occurs: %v ", cluster.Namespace, cluster.Name, err)
		return false, err
	}

	if hasChanged {
		glog.V(6).Infof("[clusterAction] cluster has changed cluster: %s/%s", cluster.Namespace, cluster.Name)
		return true, nil
	}

	// run sanity check if needed
	needSanity, err := sanitycheck.RunSanityChecks(admin, &c.config.redis, c.podControl, cluster, infos, true)
	if err != nil {
		glog.Errorf("[clusterAction] cluster %s/%s, an error occurs during sanitycheck: %v ", cluster.Namespace, cluster.Name, err)
		return false, err
	}
	if needSanity {
		glog.V(3).Infof("[clusterAction] run sanitycheck cluster: %s/%s", cluster.Namespace, cluster.Name)
		return sanitycheck.RunSanityChecks(admin, &c.config.redis, c.podControl, cluster, infos, false)
	}

	glog.V(6).Infof("[clusterAction] cluster hasn't changed cluster: %s/%s", cluster.Namespace, cluster.Name)
	return false, nil
}

// manageRollingUpdate used to manage properly a cluster rolling update if the podtemplate spec has changed
func (c *Controller) manageRollingUpdate(admin redis.AdminInterface, cluster *rapi.RedisCluster, rCluster *redis.Cluster, nodes redis.Nodes) (bool, error) {
	nbRequirePodForSpec := *cluster.Spec.NumberOfMaster * (1 + *cluster.Spec.ReplicationFactor)
	nbPodByNodeMigration := 1 + *cluster.Spec.ReplicationFactor
	nbPodToCreate := nbRequirePodForSpec + nbPodByNodeMigration - cluster.Status.Cluster.NbPods
	if nbPodToCreate > 0 {
		for i := int32(0); i < nbPodToCreate; i++ {
			_, err := c.podControl.CreatePod(cluster)
			if err != nil {
				return false, err
			}
		}
		return true, nil
	}

	// pods with new version are ready it is time to migrate slots and key
	newNodes := nodes.FilterByFunc(func(n *redis.Node) bool {
		if n.Pod == nil {
			return false
		}
		if comparePodSpec(&cluster.Spec.PodTemplate.Spec, &n.Pod.Spec) {
			return true
		}
		return false
	})
	oldNodes := nodes.FilterByFunc(func(n *redis.Node) bool {
		if n.Pod == nil {
			return false
		}
		if !comparePodSpec(&cluster.Spec.PodTemplate.Spec, &n.Pod.Spec) {
			return true
		}
		return false
	})
	newMasterNodes, newSlaveNodes, newNoneNodes := clustering.ClassifyNodesByRole(newNodes)
	oldMasterNodes, oldSlaveNodes, _ := clustering.ClassifyNodesByRole(oldNodes)

	selectedMasters, selectedNewMasters, err := clustering.SelectMastersToReplace(oldMasterNodes, newMasterNodes, newNoneNodes, *cluster.Spec.NumberOfMaster, 1)
	if err != nil {
		return false, err
	}
	currentSlaves := append(oldSlaveNodes, newSlaveNodes...)
	futurSlaves := newNoneNodes.FilterByFunc(func(n *redis.Node) bool {
		for _, newMaster := range selectedNewMasters {
			if n.ID == newMaster.ID {
				return false
			}
		}
		return true
	})

	slavesByMaster, bestEffort := clustering.PlaceSlaves(rCluster, selectedMasters, currentSlaves, futurSlaves, *cluster.Spec.ReplicationFactor)
	if bestEffort {
		cluster.Status.Cluster.NodesPlacement = rapi.NodesPlacementInfoBestEffort
	} else {
		cluster.Status.Cluster.NodesPlacement = rapi.NodesPlacementInfoOptimal
	}

	if err = clustering.AttachingSlavesToMaster(rCluster, admin, slavesByMaster); err != nil {
		glog.Error("Unable to dispatch slave on new master, err:", err)
		return false, err
	}

	currentMasters := append(oldMasterNodes, newMasterNodes...)
	allMasters := append(currentMasters, selectedNewMasters...)
	// now we can move slot from old master to new master
	if err = clustering.DispatchSlotToNewMasters(rCluster, admin, selectedMasters, currentMasters, allMasters); err != nil {
		glog.Error("Unable to dispatch slot on new master, err:", err)
		return false, err
	}

	removedMasters, removeSlaves := getOldNodesToRemove(currentMasters, selectedMasters, nodes)

	nodesToDelete, err := detachAndForgetNodes(admin, removedMasters, removeSlaves)
	if err != nil {
		glog.Error("Unable to detach and forhet old masters and associated slaves, err:", err)
		return false, err
	}

	for _, node := range nodesToDelete {
		if err := c.podControl.DeletePod(cluster, node.Pod.Name); err != nil {
			glog.Errorf("unable to delete the pod %s/%s, err:%v", node.Pod.Name, node.Pod.Namespace, err)
			return false, err
		}
	}

	return false, nil
}

// managePodScaleDown used to manage properly the scale down of a cluster
func (c *Controller) managePodScaleDown(admin redis.AdminInterface, cluster *rapi.RedisCluster, rCluster *redis.Cluster, nodes redis.Nodes) (bool, error) {
	glog.V(6).Info("managePodScaleDown START")
	defer glog.V(6).Info("managePodScaleDown STOP")

	if uselessNodes, ok := checkNoPodsUseless(cluster); !ok {
		for _, node := range uselessNodes {
			if err := c.podControl.DeletePod(cluster, node.PodName); err != nil {
				return false, err
			}
		}
	}

	if slavesOfSlave, ok := checkslaveOfSlave(cluster); !ok {
		glog.V(6).Info("checkslaveOfSlave NOT OK")
		// Currently the rest of algo didn't manage properly the fact that Redis is no able to attach a Slave to another slave.
		// So we detach them. and another part of the controller will reasign properly the Slave to the master if needed.
		var errGlobal error
		for _, slaves := range slavesOfSlave {
			for _, slave := range slaves {
				node, err := nodes.GetNodeByID(slave.ID)
				if err != nil {
					glog.Errorf("unable to found the Node with the ID: %s", slave.ID)
					errGlobal = err
					continue
				}
				if err := admin.DetachSlave(node); err != nil {
					glog.Errorf("unable to detach the Slave Node with the ID: %s", node.ID)
					errGlobal = err
					continue
				}
				if err := c.podControl.DeletePod(cluster, slave.PodName); err != nil {
					glog.Errorf("unable to delete the pod %s corresponding to the Slave Node with the ID: %s", slave.PodName, node.ID)
					errGlobal = err
					continue
				}
			}
		}
		return true, errGlobal
	}

	if slaveByMaster, ok := checkReplicationFactor(cluster); !ok {
		glog.V(6).Info("checkReplicationFactor NOT OK")
		// if not OK means that the level of replication is not good
		for idMaster, slavesID := range slaveByMaster {
			diff := int32(len(slavesID)) - *cluster.Spec.ReplicationFactor
			if diff < 0 {
				// not enough slaves on this master
				// found an avalaible redis node to be the new slaves
				selection, err := searchAvailableSlaveForMasterID(nodes, idMaster, -diff)
				if err != nil {
					return false, err
				}
				// get the master node
				master, err := nodes.GetNodeByID(idMaster)
				if err != nil {
					return false, err
				}
				errorAppends := ""
				for _, node := range selection {
					if err = admin.AttachSlaveToMaster(node, master); err != nil {
						glog.Errorf("error during manageScaleDown")
						errorAppends += fmt.Sprintf("[%s]err:%s ,", node.ID, err)
					}
				}

				if errorAppends != "" {
					return false, fmt.Errorf("unable to attach slaves on Master %s: %s", idMaster, errorAppends)
				}
			} else if diff > 0 {
				// TODO (IMP): this if can be remove since it should no happen.
				// too many slave on this master
				// select on slave to be removed
				podsToDeletion, err := selectSlavesToDelete(cluster, nodes, idMaster, slavesID, diff)
				if err != nil {
					return false, err
				}
				errs := []error{}
				for _, rNode := range podsToDeletion {
					admin.DetachSlave(rNode)
					if rNode.Pod != nil {
						if err := c.podControl.DeletePod(cluster, rNode.Pod.Name); err != nil {
							errs = append(errs, err)
						}
					}
				}
				return true, errors.NewAggregate(errs)
			}
		}
	}

	if nbMasterToDelete, ok := checkNumberOfMaster(cluster); !ok {
		glog.V(6).Info("checkNumberOfMaster NOT OK")
		newNumberOfMaster := cluster.Status.Cluster.NumberOfMaster

		// we decrease only one by one the number of master in order to limit the impact on the client.
		if nbMasterToDelete > 0 {
			newNumberOfMaster--
		}

		//First, we define the new masters
		newMasters, curMasters, allMaster, err := clustering.DispatchMasters(rCluster, nodes, newNumberOfMaster, admin)
		if err != nil {
			glog.Errorf("Error while dispatching slots to masters, err: %v", err)
			cluster.Status.Cluster.Status = rapi.ClusterStatusKO
			rCluster.Status = rapi.ClusterStatusKO
			return false, err
		}

		if err := clustering.DispatchSlotToNewMasters(rCluster, admin, newMasters, curMasters, allMaster); err != nil {
			glog.Error("Unable to dispatch slot on new master, err:", err)
			return false, err
		}

		removedMasters, removeSlaves := getOldNodesToRemove(curMasters, newMasters, nodes)

		if _, err := detachAndForgetNodes(admin, removedMasters, removeSlaves); err != nil {
			glog.Error("Unable to detach and forhet old masters and associated slaves, err:", err)
			return false, err
		}
	}

	return false, nil
}

func getOldNodesToRemove(curMasters, newMasters, nodes redis.Nodes) (removedMasters, removeSlaves redis.Nodes) {
	removedMasters = redis.Nodes{}
	for _, node := range curMasters {
		if _, err := newMasters.GetNodeByID(node.ID); err == nil {
			continue
		}
		removedMasters = append(removedMasters, node)
	}

	removeSlaves = redis.Nodes{}
	for _, master := range removedMasters {
		slaves := nodes.FilterByFunc(func(node *redis.Node) bool {
			if redis.IsSlave(node) && (node.MasterReferent == master.ID) {
				return true
			}
			return false
		})
		removeSlaves = append(removeSlaves, slaves...)
	}

	return removedMasters, removeSlaves
}

func detachAndForgetNodes(admin redis.AdminInterface, masters, slaves redis.Nodes) (redis.Nodes, error) {
	for _, node := range slaves {
		if err := admin.DetachSlave(node); err != nil {
			glog.Errorf("unable to detach the slave with ID:%s, err:%v", node.ID, err)
		}
	}

	removedNodes := append(masters, slaves...)
	for _, node := range removedNodes {
		if err := admin.ForgetNode(node.ID); err != nil {
			glog.Errorf("unable to forger the node with ID:%s, err:%v", node.ID, err)
		}
	}
	return removedNodes, nil
}

func searchAvailableSlaveForMasterID(nodes redis.Nodes, idMaster string, nbSlavedNeeded int32) (redis.Nodes, error) {
	selection := redis.Nodes{}
	var err error

	if nbSlavedNeeded <= 0 {
		return selection, err
	}

	for _, node := range nodes {
		if len(selection) == int(nbSlavedNeeded) {
			break
		}
		if redis.IsMasterWithNoSlot(node) {
			selection = append(selection, node)
		}
	}

	if len(selection) < int(nbSlavedNeeded) {
		return selection, fmt.Errorf("unable to found enough redis nodes")
	}
	return selection, err
}

func selectSlavesToDelete(cluster *rapi.RedisCluster, nodes redis.Nodes, idMaster string, slavesID []string, nbSlavesToDelete int32) (redis.Nodes, error) {
	selection := redis.Nodes{}

	masterNodeName := "default"
	for _, node := range cluster.Status.Cluster.Nodes {
		if node.ID == idMaster {
			if node.Pod != nil {
				// TODO improve this with Node labels
				masterNodeName = node.Pod.Spec.NodeName
			}
		}
	}

	secondSelection := redis.Nodes{}

	for _, slaveID := range slavesID {
		if len(selection) == int(nbSlavesToDelete) {
			return selection, nil
		}
		if slave, err := nodes.GetNodeByID(slaveID); err == nil {
			if slave.Pod.Spec.NodeName == masterNodeName {
				selection = append(selection, slave)
			} else {
				secondSelection = append(secondSelection, slave)
			}
		}
	}

	for _, node := range secondSelection {
		if len(selection) == int(nbSlavesToDelete) {
			return selection, nil
		}
		selection = append(selection, node)

	}

	if len(selection) == int(nbSlavesToDelete) {
		return selection, nil
	}

	return selection, fmt.Errorf("unable to found enough Slave to delete")
}

// applyConfiguration apply new configuration if needed:
// - add or delete pods
// - configure the redis-server process
func (c *Controller) applyConfiguration(admin redis.AdminInterface, cluster *rapi.RedisCluster) (bool, error) {
	glog.V(6).Info("applyConfiguration START")
	defer glog.V(6).Info("applyConfiguration STOP")

	asChanged := false

	// Configuration
	cReplicaFactor := *cluster.Spec.ReplicationFactor
	cNbMaster := *cluster.Spec.NumberOfMaster

	rCluster, nodes, err := newRedisCluster(admin, cluster)
	if err != nil {
		glog.Errorf("Unable to create the RedisCluster view, error:%v", err)
		return false, err
	}

	if needRollingUpdate(cluster) {
		if setRollingUpdategCondition(&cluster.Status, true) {
			if cluster, err = c.updateHandler(cluster); err != nil {
				return false, err
			}
		}

		glog.Info("applyConfiguration needRollingUpdate")
		return c.manageRollingUpdate(admin, cluster, rCluster, nodes)
	}
	if setRollingUpdategCondition(&cluster.Status, false) {
		if cluster, err = c.updateHandler(cluster); err != nil {
			return false, err
		}
	}

	if needLessPods(cluster) {
		if setRebalancingCondition(&cluster.Status, true) {
			if cluster, err = c.updateHandler(cluster); err != nil {
				return false, err
			}
		}
		glog.Info("applyConfiguration needLessPods")
		return c.managePodScaleDown(admin, cluster, rCluster, nodes)
	}
	if setRebalancingCondition(&cluster.Status, false) {
		if cluster, err = c.updateHandler(cluster); err != nil {
			return false, err
		}
	}

	clusterStatus := &cluster.Status.Cluster
	if (clusterStatus.NbPods - clusterStatus.NbRedisRunning) != 0 {
		glog.V(3).Infof("All pods not ready wait to be ready, nbPods: %d, nbPodsReady: %d", clusterStatus.NbPods, clusterStatus.NbRedisRunning)
		return false, err
	}

	//First, we define the new masters
	newMasters, curMasters, allMaster, err := clustering.DispatchMasters(rCluster, nodes, cNbMaster, admin)
	if err != nil {
		glog.Errorf("Cannot dispatch slots to masters: %v", err)
		rCluster.Status = rapi.ClusterStatusKO
		return false, err
	}
	if len(newMasters) != len(curMasters) {
		asChanged = true
	}

	// Second select Node that is already a slave
	currentSlaveNodes := nodes.FilterByFunc(redis.IsSlave)

	//New slaves are slaves which is currently a master with no slots
	newSlave := nodes.FilterByFunc(func(nodeA *redis.Node) bool {
		for _, nodeB := range newMasters {
			if nodeA.ID == nodeB.ID {
				return false
			}
		}
		for _, nodeB := range currentSlaveNodes {
			if nodeA.ID == nodeB.ID {
				return false
			}
		}
		return true
	})

	// Depending on whether we scale up or down, we will dispatch slaves before/after the dispatch of slots
	if cNbMaster < int32(len(curMasters)) {
		// this happens usually after a scale down of the cluster
		// we should dispatch slots before dispatching slaves
		if err := clustering.DispatchSlotToNewMasters(rCluster, admin, newMasters, curMasters, allMaster); err != nil {
			glog.Error("Unable to dispatch slot on new master, err:", err)
			return false, err
		}

		// assign master/slave roles
		newRedisSlavesByMaster, bestEffort := clustering.PlaceSlaves(rCluster, newMasters, currentSlaveNodes, newSlave, cReplicaFactor)
		if bestEffort {
			rCluster.NodesPlacement = rapi.NodesPlacementInfoBestEffort
		}

		if err := clustering.AttachingSlavesToMaster(rCluster, admin, newRedisSlavesByMaster); err != nil {
			glog.Error("Unable to dispatch slave on new master, err:", err)
			return false, err
		}
	} else {
		//We are scaling up the nbmaster or the nbmaster doesn't change.
		// assign master/slave roles
		newRedisSlavesByMaster, bestEffort := clustering.PlaceSlaves(rCluster, newMasters, currentSlaveNodes, newSlave, cReplicaFactor)
		if bestEffort {
			rCluster.NodesPlacement = rapi.NodesPlacementInfoBestEffort
		}

		if err := clustering.AttachingSlavesToMaster(rCluster, admin, newRedisSlavesByMaster); err != nil {
			glog.Error("Unable to dispatch slave on new master, err:", err)
			return false, err
		}

		if err := clustering.DispatchSlotToNewMasters(rCluster, admin, newMasters, curMasters, allMaster); err != nil {
			glog.Error("Unable to dispatch slot on new master, err:", err)
			return false, err
		}
	}

	glog.V(4).Infof("new nodes status: \n %v", nodes)

	rCluster.Status = rapi.ClusterStatusOK
	// wait a bit for the cluster to propagate configuration to reduce warning logs because of temporary inconsistency
	time.Sleep(1 * time.Second)
	return asChanged, nil
}

func newRedisCluster(admin redis.AdminInterface, cluster *rapi.RedisCluster) (*redis.Cluster, redis.Nodes, error) {
	infos, err := admin.GetClusterInfos()
	if redis.IsPartialError(err) {
		glog.Errorf("Error getting consolidated view of the cluster err: %v", err)
		return nil, nil, err
	}

	// now we can trigger the rebalance
	nodes := infos.GetNodes()

	// build redis cluster vision
	rCluster := &redis.Cluster{
		Name:      cluster.Name,
		Namespace: cluster.Namespace,
		Nodes:     make(map[string]*redis.Node),
	}

	for _, node := range nodes {
		rCluster.Nodes[node.ID] = node
	}

	for _, node := range cluster.Status.Cluster.Nodes {
		if rNode, ok := rCluster.Nodes[node.ID]; ok {
			rNode.Pod = node.Pod
		}
	}

	return rCluster, nodes, nil
}
