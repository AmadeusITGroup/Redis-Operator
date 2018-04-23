package sanitycheck

import (
	"time"

	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/util/errors"

	rapi "github.com/amadeusitgroup/redis-operator/pkg/api/redis/v1"
	"github.com/amadeusitgroup/redis-operator/pkg/controller/pod"
)

// FixTerminatingPods used to for the deletion of pod blocked in terminating status.
// in it append the this method will for the deletion of the Pod.
func FixTerminatingPods(cluster *rapi.RedisCluster, podControl pod.RedisClusterControlInteface, maxDuration time.Duration, dryRun bool) (bool, error) {
	var errs []error
	var actionDone bool

	if maxDuration == time.Duration(0) {
		return actionDone, nil
	}

	currentPods, err := podControl.GetRedisClusterPods(cluster)
	if err != nil {
		glog.Errorf("unable to retrieve the Pod list, err:%v", err)
	}

	now := time.Now()
	for _, pod := range currentPods {
		if pod.DeletionTimestamp == nil {
			// ignore pod without deletion timestamp
			continue
		}
		maxTime := pod.DeletionTimestamp.Add(maxDuration) // adding MaxDuration for configuration
		if maxTime.Before(now) {
			actionDone = true
			// it means that this pod should already been deleted since a wild
			if !dryRun {
				if err := podControl.DeletePod(cluster, pod.Name); err != nil {
					errs = append(errs, err)
				}
			}
		}
	}

	return actionDone, errors.NewAggregate(errs)
}
