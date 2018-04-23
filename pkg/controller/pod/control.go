package pod

import (
	"fmt"

	kapiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/record"

	rapi "github.com/amadeusitgroup/redis-operator/pkg/api/redis/v1"
	"github.com/golang/glog"
)

// RedisClusterControlInteface interface for the RedisClusterPodControl
type RedisClusterControlInteface interface {
	// GetRedisClusterPods return list of Pod attached to a RedisCluster
	GetRedisClusterPods(redisCluster *rapi.RedisCluster) ([]*kapiv1.Pod, error)
	// CreatePod used to create a Pod from the RedisCluster pod template
	CreatePod(redisCluster *rapi.RedisCluster) (*kapiv1.Pod, error)
	// DeletePod used to delete a pod from its name
	DeletePod(redisCluster *rapi.RedisCluster, podName string) error
	// DeletePodNow used to delete now (force) a pod from its name
	DeletePodNow(redisCluster *rapi.RedisCluster, podName string) error
}

var _ RedisClusterControlInteface = &RedisClusterControl{}

// RedisClusterControl contains requieres accessor to managing the RedisCluster pods
type RedisClusterControl struct {
	PodLister  corev1listers.PodLister
	KubeClient clientset.Interface
	Recorder   record.EventRecorder
}

// NewRedisClusterControl builds and returns new NewRedisClusterControl instance
func NewRedisClusterControl(lister corev1listers.PodLister, client clientset.Interface, rec record.EventRecorder) *RedisClusterControl {
	ctrl := &RedisClusterControl{
		PodLister:  lister,
		KubeClient: client,
		Recorder:   rec,
	}
	return ctrl
}

// GetRedisClusterPods return list of Pod attached to a RedisCluster
func (p *RedisClusterControl) GetRedisClusterPods(redisCluster *rapi.RedisCluster) ([]*kapiv1.Pod, error) {
	selector, err := CreateRedisClusterLabelSelector(redisCluster)
	if err != nil {
		return nil, err
	}
	return p.PodLister.List(selector)
}

// CreatePod used to create a Pod from the RedisCluster pod template
func (p *RedisClusterControl) CreatePod(redisCluster *rapi.RedisCluster) (*kapiv1.Pod, error) {
	pod, err := initPod(redisCluster)
	if err != nil {
		return pod, err
	}
	glog.V(6).Infof("CreatePod: %s/%s", redisCluster.Namespace, pod.Name)
	return p.KubeClient.Core().Pods(redisCluster.Namespace).Create(pod)
}

// DeletePod used to delete a pod from its name
func (p *RedisClusterControl) DeletePod(redisCluster *rapi.RedisCluster, podName string) error {
	glog.V(6).Infof("DeletePod: %s/%s", redisCluster.Namespace, podName)
	return p.deletePodGracefullperiode(redisCluster, podName, nil)
}

// DeletePodNow used to delete now (force) a pod from its name
func (p *RedisClusterControl) DeletePodNow(redisCluster *rapi.RedisCluster, podName string) error {
	glog.V(6).Infof("DeletePod: %s/%s", redisCluster.Namespace, podName)
	now := int64(0)
	return p.deletePodGracefullperiode(redisCluster, podName, &now)
}

// DeletePodNow used to delete now (force) a pod from its name
func (p *RedisClusterControl) deletePodGracefullperiode(redisCluster *rapi.RedisCluster, podName string, period *int64) error {
	return p.KubeClient.Core().Pods(redisCluster.Namespace).Delete(podName, &metav1.DeleteOptions{GracePeriodSeconds: period})
}

func initPod(redisCluster *rapi.RedisCluster) (*kapiv1.Pod, error) {
	desiredLabels, err := GetLabelsSet(redisCluster)
	if err != nil {
		return nil, err
	}
	desiredAnnotations, err := GetAnnotationsSet(redisCluster)
	if err != nil {
		return nil, err
	}
	PodName := fmt.Sprintf("rediscluster-%s-", redisCluster.Name)
	pod := &kapiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          desiredLabels,
			Annotations:     desiredAnnotations,
			GenerateName:    PodName,
			OwnerReferences: []metav1.OwnerReference{BuildOwnerReference(redisCluster)},
		},
	}

	pod.Spec = *redisCluster.Spec.PodTemplate.Spec.DeepCopy()

	return pod, nil
}

// BuildOwnerReference used to build the OwnerReference from a RedisCluster
func BuildOwnerReference(cluster *rapi.RedisCluster) metav1.OwnerReference {
	controllerRef := metav1.OwnerReference{
		APIVersion:         rapi.SchemeGroupVersion.String(),
		Kind:               rapi.ResourceKind,
		Name:               cluster.Name,
		UID:                cluster.UID,
		BlockOwnerDeletion: boolPtr(true),
		Controller:         boolPtr(true),
	}

	return controllerRef
}

func boolPtr(value bool) *bool {
	return &value
}
