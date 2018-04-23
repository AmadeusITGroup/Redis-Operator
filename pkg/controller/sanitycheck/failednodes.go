package sanitycheck

import (
	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/util/errors"

	rapi "github.com/amadeusitgroup/redis-operator/pkg/api/redis/v1"
	"github.com/amadeusitgroup/redis-operator/pkg/redis"
)

// FixFailedNodes fix failed nodes: in some cases (cluster without enough master after crash or scale down), some nodes may still know about fail nodes
func FixFailedNodes(admin redis.AdminInterface, cluster *rapi.RedisCluster, infos *redis.ClusterInfos, dryRun bool) (bool, error) {
	forgetSet := listGhostNodes(cluster, infos)
	var errs []error
	doneAnAction := false
	for id := range forgetSet {
		doneAnAction = true
		glog.Infof("Sanitychecks: Forgetting failed node %s, this command might fail, this is not an error", id)
		if !dryRun {
			if err := admin.ForgetNode(id); err != nil {
				errs = append(errs, err)
			}
		}
	}

	return doneAnAction, errors.NewAggregate(errs)
}

// listGhostNodes : A Ghost node is a node still known by some redis node but which doesn't exists anymore
// meaning it is failed, and pod not in kubernetes, or without targetable IP
func listGhostNodes(cluster *rapi.RedisCluster, infos *redis.ClusterInfos) map[string]bool {
	ghostNodesSet := map[string]bool{}
	if infos == nil || infos.Infos == nil {
		return ghostNodesSet
	}
	for _, nodeinfos := range infos.Infos {
		for _, node := range nodeinfos.Friends {
			// only forget it when no more part of kubernetes, or if noaddress
			if node.HasStatus(redis.NodeStatusNoAddr) {
				ghostNodesSet[node.ID] = true
			}
			if node.HasStatus(redis.NodeStatusFail) || node.HasStatus(redis.NodeStatusPFail) {
				// only forget it when no more part of kubernetes, or if noaddress
				found := false
				for _, pod := range cluster.Status.Cluster.Nodes {
					if pod.ID == node.ID {
						found = true
					}
				}
				if !found {
					ghostNodesSet[node.ID] = true
				}
			}
		}
	}
	return ghostNodesSet
}
