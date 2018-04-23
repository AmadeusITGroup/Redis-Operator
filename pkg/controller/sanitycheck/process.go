package sanitycheck

import (
	"time"

	"github.com/golang/glog"

	rapi "github.com/amadeusitgroup/redis-operator/pkg/api/redis/v1"
	"github.com/amadeusitgroup/redis-operator/pkg/config"
	"github.com/amadeusitgroup/redis-operator/pkg/controller/pod"
	"github.com/amadeusitgroup/redis-operator/pkg/redis"
)

// RunSanityChecks function used to run all the sanity check on the current cluster
// Return actionDone = true if a modification has been made on the cluster
func RunSanityChecks(admin redis.AdminInterface, config *config.Redis, podControl pod.RedisClusterControlInteface, cluster *rapi.RedisCluster, infos *redis.ClusterInfos, dryRun bool) (actionDone bool, err error) {
	// fix node view inconsistency for which nodes know other nodes
	if actionDone, err = FixNodesNotMeet(admin, infos, dryRun); err != nil {
		return actionDone, err
	} else if actionDone {
		glog.V(2).Infof("FixNodesNotMeet done an action on the cluster (dryRun:%v)", dryRun)
		return actionDone, nil
	}

	// * fix failed nodes: in some cases (cluster without enough master after crash or scale down), some nodes may still know about fail nodes
	if actionDone, err = FixFailedNodes(admin, cluster, infos, dryRun); err != nil {
		return actionDone, err
	} else if actionDone {
		glog.V(2).Infof("FixFailedNodes done an action on the cluster (dryRun:%v)", dryRun)
		return actionDone, nil
	}

	// forget nodes and delete pods when a redis node is untrusted.
	if actionDone, err = FixUntrustedNodes(admin, podControl, cluster, infos, dryRun); err != nil {
		return actionDone, err
	} else if actionDone {
		glog.V(2).Infof("FixUntrustedNodes done an action on the cluster (dryRun:%v)", dryRun)
		return actionDone, nil
	}

	// forget nodes and delete pods when a redis node is untrusted.
	if actionDone, err = FixTerminatingPods(cluster, podControl, 5*time.Minute, dryRun); err != nil {
		return actionDone, err
	} else if actionDone {
		glog.V(2).Infof("FixTerminatingPods done an action on the cluster (dryRun:%v)", dryRun)
		return actionDone, nil
	}

	// forget nodes and delete pods when a redis node is untrusted.
	if actionDone, err = FixClusterSplit(admin, config, infos, dryRun); err != nil {
		return actionDone, err
	} else if actionDone {
		glog.V(2).Infof("FixClusterSplit done an action on the cluster (dryRun:%v)", dryRun)
		return actionDone, nil
	}

	return actionDone, err
}
