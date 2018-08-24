package sanitycheck

import (
	"github.com/golang/glog"

	kapi "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/errors"

	rapi "github.com/amadeusitgroup/redis-operator/pkg/api/redis/v1"
	"github.com/amadeusitgroup/redis-operator/pkg/controller/pod"
	"github.com/amadeusitgroup/redis-operator/pkg/redis"
)

// FixUntrustedNodes used to remove Nodes that are not trusted by other nodes. It can append when a node
// are removed from the cluster (with the "forget nodes" command) but try to rejoins the cluster.
func FixUntrustedNodes(admin redis.AdminInterface, podControl pod.RedisClusterControlInteface, cluster *rapi.RedisCluster, infos *redis.ClusterInfos, dryRun bool) (bool, error) {
	untrustedNode := listUntrustedNodes(infos)
	var errs []error
	doneAnAction := false

	currentPods, err := podControl.GetRedisClusterPods(cluster)
	if err != nil {
		glog.Errorf("unable to retrieve the Pod list, err:%v", err)
	}

	for id, uNode := range untrustedNode {
		getByIPFunc := func(n *redis.Node) bool {
			if n.IP == uNode.IP && n.ID != uNode.ID {
				return true
			}
			return false
		}
		node2, err := infos.GetNodes().GetNodesByFunc(getByIPFunc)
		if err != nil && !redis.IsNodeNotFoundedError(err) {
			glog.Errorf("Error with GetNodesByFunc(getByIPFunc) search function")
			errs = append(errs, err)
			continue
		}
		if len(node2) > 0 {
			// it means the POD is used by another Redis node ID so we should not delete the pod.
			continue
		}
		exist, reused := checkIfPodNameExistAndIsReused(uNode, currentPods)
		if exist && !reused {
			if err := podControl.DeletePod(cluster, uNode.Pod.Name); err != nil {
				errs = append(errs, err)
			}
		}
		doneAnAction = true
		if !dryRun {
			if err := admin.ForgetNode(id); err != nil {
				errs = append(errs, err)
			}
		}
	}

	return doneAnAction, errors.NewAggregate(errs)
}

func listUntrustedNodes(infos *redis.ClusterInfos) map[string]*redis.Node {
	untrustedNodes := make(map[string]*redis.Node)
	if infos == nil || infos.Infos == nil {
		return untrustedNodes
	}
	for _, nodeinfos := range infos.Infos {
		for _, node := range nodeinfos.Friends {
			// only forget it when no more part of kubernetes, or if noaddress
			if node.HasStatus(redis.NodeStatusHandshake) {
				if _, found := untrustedNodes[node.ID]; !found {
					untrustedNodes[node.ID] = node
				}
			}
		}
	}
	return untrustedNodes
}

func checkIfPodNameExistAndIsReused(node *redis.Node, podlist []*kapi.Pod) (exist bool, reused bool) {
	if node.Pod == nil {
		return exist, reused
	}
	for _, currentPod := range podlist {
		if currentPod.Name == node.Pod.Name {
			exist = true
			if currentPod.Status.PodIP == node.Pod.Status.PodIP {
				// this check is use to see if the Pod name is not use by another RedisNode.
				// for that we check the the Pod name from the Redis node is not used by another
				// Redis node, by comparing the IP of the current Pod with the Pod from the cluster bom.
				// if the Pod  IP and Name from the redis info is equal to the IP/NAME from the getPod; it
				// means that the Pod is still use and the Redis Node is not a ghost
				reused = true
				break
			}
		}
	}
	return exist, reused
}
