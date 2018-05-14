package controller

import (
	"fmt"

	kapiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"

	rapi "github.com/amadeusitgroup/redis-operator/pkg/api/redis/v1"
	"github.com/amadeusitgroup/redis-operator/pkg/controller/pod"
)

// ServicesControlInterface inferface for the ServicesControl
type ServicesControlInterface interface {
	// CreateRedisClusterService used to create the Kubernetes Service needed to access the Redis Cluster
	CreateRedisClusterService(redisCluster *rapi.RedisCluster) (*kapiv1.Service, error)
	// DeleteRedisClusterService used to delete the Kubernetes Service linked to the Redis Cluster
	DeleteRedisClusterService(redisCluster *rapi.RedisCluster) error
	// GetRedisClusterService used to retrieve the Kubernetes Service associated to the RedisCluster
	GetRedisClusterService(redisCluster *rapi.RedisCluster) (*kapiv1.Service, error)
}

// ServicesControl contains all information for managing Kube Services
type ServicesControl struct {
	KubeClient clientset.Interface
	Recorder   record.EventRecorder
}

// NewServicesControl builds and returns new ServicesControl instance
func NewServicesControl(client clientset.Interface, rec record.EventRecorder) *ServicesControl {
	ctrl := &ServicesControl{
		KubeClient: client,
		Recorder:   rec,
	}

	return ctrl
}

// GetRedisClusterService used to retrieve the Kubernetes Service associated to the RedisCluster
func (s *ServicesControl) GetRedisClusterService(redisCluster *rapi.RedisCluster) (*kapiv1.Service, error) {
	serviceName := getServiceName(redisCluster)
	svc, err := s.KubeClient.CoreV1().Services(redisCluster.Namespace).Get(serviceName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if svc.Name != serviceName {
		return nil, fmt.Errorf("Couldn't find service named %s", serviceName)
	}
	return svc, nil
}

// CreateRedisClusterService used to create the Kubernetes Service needed to access the Redis Cluster
func (s *ServicesControl) CreateRedisClusterService(redisCluster *rapi.RedisCluster) (*kapiv1.Service, error) {
	serviceName := getServiceName(redisCluster)
	desiredlabels, err := pod.GetLabelsSet(redisCluster)
	if err != nil {
		return nil, err

	}

	desiredAnnotations, err := pod.GetAnnotationsSet(redisCluster)
	if err != nil {
		return nil, err
	}
	newService := &kapiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          desiredlabels,
			Annotations:     desiredAnnotations,
			Name:            serviceName,
			OwnerReferences: []metav1.OwnerReference{pod.BuildOwnerReference(redisCluster)},
		},
		Spec: kapiv1.ServiceSpec{
			ClusterIP: "None",
			Ports:     []kapiv1.ServicePort{{Port: 6379, Name: "redis"}},
			Selector:  desiredlabels,
		},
	}
	return s.KubeClient.CoreV1().Services(redisCluster.Namespace).Create(newService)
}

// DeleteRedisClusterService used to delete the Kubernetes Service linked to the Redis Cluster
func (s *ServicesControl) DeleteRedisClusterService(redisCluster *rapi.RedisCluster) error {
	serviceName := getServiceName(redisCluster)
	return s.KubeClient.CoreV1().Services(redisCluster.Namespace).Delete(serviceName, nil)
}

func getServiceName(redisCluster *rapi.RedisCluster) string {
	serviceName := redisCluster.Name
	if redisCluster.Spec.ServiceName != "" {
		serviceName = redisCluster.Spec.ServiceName
	}
	return serviceName
}
