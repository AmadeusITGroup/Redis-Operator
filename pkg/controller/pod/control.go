package pod

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

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
	return p.PodLister.Pods(redisCluster.Namespace).List(selector)
}

// CreatePod used to create a Pod from the RedisCluster pod template
func (p *RedisClusterControl) CreatePod(redisCluster *rapi.RedisCluster) (*kapiv1.Pod, error) {
	pod, err := initPod(redisCluster)
	if err != nil {
		return pod, err
	}
	glog.V(6).Infof("CreatePod: %s/%s", redisCluster.Namespace, pod.Name)
	return p.KubeClient.CoreV1().Pods(redisCluster.Namespace).Create(pod)
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
	return p.KubeClient.CoreV1().Pods(redisCluster.Namespace).Delete(podName, &metav1.DeleteOptions{GracePeriodSeconds: period})
}

func initPod(redisCluster *rapi.RedisCluster) (*kapiv1.Pod, error) {
	if redisCluster == nil {
		return nil, fmt.Errorf("rediscluster nil pointer")
	}

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
			Namespace:       redisCluster.Namespace,
			Labels:          desiredLabels,
			Annotations:     desiredAnnotations,
			GenerateName:    PodName,
			OwnerReferences: []metav1.OwnerReference{BuildOwnerReference(redisCluster)},
		},
	}

	if redisCluster.Spec.PodTemplate == nil {
		return nil, fmt.Errorf("rediscluster[%s/%s] PodTemplate missing", redisCluster.Namespace, redisCluster.Name)
	}
	pod.Spec = *redisCluster.Spec.PodTemplate.Spec.DeepCopy()

	// Generate a MD5 representing the PodSpec send
	hash, err := GenerateMD5Spec(&pod.Spec)
	if err != nil {
		return nil, err
	}
	pod.Annotations[rapi.PodSpecMD5LabelKey] = hash

	return pod, nil
}

// GenerateMD5Spec used to generate the PodSpec MD5 hash
func GenerateMD5Spec(spec *kapiv1.PodSpec) (string, error) {
	b, err := json.Marshal(spec)
	if err != nil {
		return "", err
	}
	hash := md5.New()
	io.Copy(hash, bytes.NewReader(b))
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// BuildOwnerReference used to build the OwnerReference from a RedisCluster
func BuildOwnerReference(cluster *rapi.RedisCluster) metav1.OwnerReference {
	controllerRef := metav1.OwnerReference{
		APIVersion: rapi.SchemeGroupVersion.String(),
		Kind:       rapi.ResourceKind,
		Name:       cluster.Name,
		UID:        cluster.UID,
		Controller: boolPtr(true),
	}

	return controllerRef
}

func boolPtr(value bool) *bool {
	return &value
}
