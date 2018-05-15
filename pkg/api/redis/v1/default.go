package v1

import (
	kapiv1 "k8s.io/api/core/v1"
)

// IsRedisClusterDefaulted check if the RedisCluster is already defaulted
func IsRedisClusterDefaulted(rc *RedisCluster) bool {
	if rc.Spec.NumberOfMaster == nil {
		return false
	}
	if rc.Spec.ReplicationFactor == nil {
		return false
	}
	return true
}

// DefaultRedisCluster defaults RedisCluster
func DefaultRedisCluster(undefaultRedisCluster *RedisCluster) *RedisCluster {
	rc := undefaultRedisCluster.DeepCopy()
	if rc.Spec.NumberOfMaster == nil {
		rc.Spec.NumberOfMaster = NewInt32(3)
	}
	if rc.Spec.ReplicationFactor == nil {
		rc.Spec.ReplicationFactor = NewInt32(1)
	}

	if rc.Spec.PodTemplate == nil {
		rc.Spec.PodTemplate = &kapiv1.PodTemplateSpec{}
	}

	rc.Status.Cluster.NumberOfMaster = 0
	rc.Status.Cluster.MinReplicationFactor = 0
	rc.Status.Cluster.MaxReplicationFactor = 0
	rc.Status.Cluster.NbPods = 0
	rc.Status.Cluster.NbPodsReady = 0
	rc.Status.Cluster.NbRedisRunning = 0

	return rc
}

// NewInt32 use to instanciate a int32 pointer
func NewInt32(val int32) *int32 {
	output := new(int32)
	*output = val

	return output
}
