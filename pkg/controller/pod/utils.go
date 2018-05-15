package pod

import (
	"fmt"
	"k8s.io/apimachinery/pkg/labels"

	rapi "github.com/amadeusitgroup/redis-operator/pkg/api/redis/v1"
)

// GetLabelsSet return labels associated to the redis-node pods
func GetLabelsSet(rediscluster *rapi.RedisCluster) (labels.Set, error) {
	desiredLabels := labels.Set{}
	if rediscluster == nil {
		return desiredLabels, fmt.Errorf("redisluster nil pointer")
	}
	if rediscluster.Spec.AdditionalLabels != nil {
		desiredLabels = rediscluster.Spec.AdditionalLabels
	}
	if rediscluster.Spec.PodTemplate != nil {
		for k, v := range rediscluster.Spec.PodTemplate.Labels {
			desiredLabels[k] = v
		}
	}
	desiredLabels[rapi.ClusterNameLabelKey] = rediscluster.Name // add rediscluster name to the Pod labels
	return desiredLabels, nil
}

// CreateRedisClusterLabelSelector creates label selector to select the jobs related to a rediscluster, stepName
func CreateRedisClusterLabelSelector(rediscluster *rapi.RedisCluster) (labels.Selector, error) {
	set, err := GetLabelsSet(rediscluster)
	if err != nil {
		return nil, err
	}
	return labels.SelectorFromSet(set), nil
}

// GetAnnotationsSet return a labels.Set of annotation from the RedisCluster
func GetAnnotationsSet(rediscluster *rapi.RedisCluster) (labels.Set, error) {
	desiredAnnotations := make(labels.Set)
	for k, v := range rediscluster.Annotations {
		desiredAnnotations[k] = v
	}

	// TODO: add createdByRef
	return desiredAnnotations, nil // no error for the moment, when we'll add createdByRef an error could be returned
}
