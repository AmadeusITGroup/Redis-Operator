package controller

import (
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rapi "github.com/amadeusitgroup/redis-operator/pkg/api/redis/v1"
)

// newCondition return a new defaulted instance of a RedisClusterCondition
func newCondition(conditionType rapi.RedisClusterConditionType, status apiv1.ConditionStatus, now metav1.Time, reason, message string) rapi.RedisClusterCondition {
	return rapi.RedisClusterCondition{
		Type:               conditionType,
		Status:             status,
		LastProbeTime:      now,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            message,
	}
}

// updateCondition return an updated version of the RedisClusterCondition
func updateCondition(from rapi.RedisClusterCondition, status apiv1.ConditionStatus, now metav1.Time, reason, message string) rapi.RedisClusterCondition {
	newCondition := from.DeepCopy()
	newCondition.LastProbeTime = now
	newCondition.Message = message
	newCondition.Reason = reason
	if status != newCondition.Status {
		newCondition.Status = status
		newCondition.LastTransitionTime = now
	}

	return *newCondition
}

func setCondition(clusterStatus *rapi.RedisClusterStatus, conditionType rapi.RedisClusterConditionType, status apiv1.ConditionStatus, now metav1.Time, reason, message string) bool {
	updated := false
	found := false
	for i, c := range clusterStatus.Conditions {
		if c.Type == conditionType {
			found = true
			if c.Status != status {
				updated = true
				clusterStatus.Conditions[i] = updateCondition(c, status, now, reason, message)
			}
		}
	}
	if !found {
		updated = true
		clusterStatus.Conditions = append(clusterStatus.Conditions, newCondition(conditionType, status, now, reason, message))
	}
	return updated
}

func setScalingCondition(clusterStatus *rapi.RedisClusterStatus, status bool) bool {
	statusCondition := apiv1.ConditionFalse
	if status {
		statusCondition = apiv1.ConditionTrue
	}
	return setCondition(clusterStatus, rapi.RedisClusterScaling, statusCondition, metav1.Now(), "cluster needs more pods", "cluster needs more pods")
}

func setRebalancingCondition(clusterStatus *rapi.RedisClusterStatus, status bool) bool {
	statusCondition := apiv1.ConditionFalse
	if status {
		statusCondition = apiv1.ConditionTrue
	}
	return setCondition(clusterStatus, rapi.RedisClusterRebalancing, statusCondition, metav1.Now(), "topology as changed", "reconfigure on-going after topology changed")
}

func setRollingUpdategCondition(clusterStatus *rapi.RedisClusterStatus, status bool) bool {
	statusCondition := apiv1.ConditionFalse
	if status {
		statusCondition = apiv1.ConditionTrue
	}
	return setCondition(clusterStatus, rapi.RedisClusterRollingUpdate, statusCondition, metav1.Now(), "Rolling update ongoing", "a Rolling update is ongoing")
}

func setClusterStatusCondition(clusterStatus *rapi.RedisClusterStatus, status bool) bool {
	statusCondition := apiv1.ConditionFalse
	if status {
		statusCondition = apiv1.ConditionTrue
	}
	return setCondition(clusterStatus, rapi.RedisClusterOK, statusCondition, metav1.Now(), "redis-cluster is correctly configure", "redis-cluster is correctly configure")
}
