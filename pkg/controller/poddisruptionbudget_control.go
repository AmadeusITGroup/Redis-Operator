package controller

import (
	"fmt"

	policyv1 "k8s.io/api/policy/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"

	rapi "github.com/amadeusitgroup/redis-operator/pkg/api/redis/v1"
	"github.com/amadeusitgroup/redis-operator/pkg/controller/pod"
)

// PodDisruptionBudgetsControlInterface inferface for the PodDisruptionBudgetsControl
type PodDisruptionBudgetsControlInterface interface {
	// CreateRedisClusterPodDisruptionBudget used to create the Kubernetes PodDisruptionBudget needed to access the Redis Cluster
	CreateRedisClusterPodDisruptionBudget(redisCluster *rapi.RedisCluster) (*policyv1.PodDisruptionBudget, error)
	// DeleteRedisClusterPodDisruptionBudget used to delete the Kubernetes PodDisruptionBudget linked to the Redis Cluster
	DeleteRedisClusterPodDisruptionBudget(redisCluster *rapi.RedisCluster) error
	// GetRedisClusterPodDisruptionBudget used to retrieve the Kubernetes PodDisruptionBudget associated to the RedisCluster
	GetRedisClusterPodDisruptionBudget(redisCluster *rapi.RedisCluster) (*policyv1.PodDisruptionBudget, error)
}

// PodDisruptionBudgetsControl contains all information for managing Kube PodDisruptionBudgets
type PodDisruptionBudgetsControl struct {
	KubeClient clientset.Interface
	Recorder   record.EventRecorder
}

// NewPodDisruptionBudgetsControl builds and returns new PodDisruptionBudgetsControl instance
func NewPodDisruptionBudgetsControl(client clientset.Interface, rec record.EventRecorder) *PodDisruptionBudgetsControl {
	ctrl := &PodDisruptionBudgetsControl{
		KubeClient: client,
		Recorder:   rec,
	}

	return ctrl
}

// GetRedisClusterPodDisruptionBudget used to retrieve the Kubernetes PodDisruptionBudget associated to the RedisCluster
func (s *PodDisruptionBudgetsControl) GetRedisClusterPodDisruptionBudget(redisCluster *rapi.RedisCluster) (*policyv1.PodDisruptionBudget, error) {
	PodDisruptionBudgetName := redisCluster.Name
	pdb, err := s.KubeClient.PolicyV1beta1().PodDisruptionBudgets(redisCluster.Namespace).Get(PodDisruptionBudgetName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if pdb.Name != PodDisruptionBudgetName {
		return nil, fmt.Errorf("Couldn't find PodDisruptionBudget named %s", PodDisruptionBudgetName)
	}
	return pdb, nil
}

// DeleteRedisClusterPodDisruptionBudget used to delete the Kubernetes PodDisruptionBudget linked to the Redis Cluster
func (s *PodDisruptionBudgetsControl) DeleteRedisClusterPodDisruptionBudget(redisCluster *rapi.RedisCluster) error {
	return s.KubeClient.PolicyV1beta1().PodDisruptionBudgets(redisCluster.Namespace).Delete(redisCluster.Name, nil)
}

// CreateRedisClusterPodDisruptionBudget used to create the Kubernetes PodDisruptionBudget needed to access the Redis Cluster
func (s *PodDisruptionBudgetsControl) CreateRedisClusterPodDisruptionBudget(redisCluster *rapi.RedisCluster) (*policyv1.PodDisruptionBudget, error) {
	PodDisruptionBudgetName := redisCluster.Name
	desiredlabels, err := pod.GetLabelsSet(redisCluster)
	if err != nil {
		return nil, err

	}

	desiredAnnotations, err := pod.GetAnnotationsSet(redisCluster)
	if err != nil {
		return nil, err
	}
	maxUnavailable := intstr.FromInt(1)
	labelSelector := metav1.LabelSelector{
		MatchLabels: desiredlabels,
	}
	newPodDisruptionBudget := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          desiredlabels,
			Annotations:     desiredAnnotations,
			Name:            PodDisruptionBudgetName,
			OwnerReferences: []metav1.OwnerReference{pod.BuildOwnerReference(redisCluster)},
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			MaxUnavailable: &maxUnavailable,
			Selector:       &labelSelector,
		},
	}
	return s.KubeClient.PolicyV1beta1().PodDisruptionBudgets(redisCluster.Namespace).Create(newPodDisruptionBudget)
}
