package garbagecollector

import (
	"fmt"
	"path"
	"time"

	"github.com/golang/glog"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	kclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	rapi "github.com/amadeusitgroup/redis-operator/pkg/api/redis/v1"
	rclientset "github.com/amadeusitgroup/redis-operator/pkg/client/clientset/versioned"
	rinformers "github.com/amadeusitgroup/redis-operator/pkg/client/informers/externalversions"
	rlisters "github.com/amadeusitgroup/redis-operator/pkg/client/listers/redis/v1"
)

const (
	// Interval represent the interval to run Garabge Collection
	Interval time.Duration = 5 * time.Second
)

// Interface GarbageCollector interface
type Interface interface {
	CollectRedisClusterGarbage() error
	InformerSync() cache.InformerSynced
}

var _ Interface = &GarbageCollector{}

// GarbageCollector represents a Workflow Garbage Collector.
// It collects orphaned Jobs
type GarbageCollector struct {
	kubeClient kclientset.Interface
	rcClient   rclientset.Interface
	rcLister   rlisters.RedisClusterLister
	rcSynced   cache.InformerSynced
}

// NewGarbageCollector builds initializes and returns a GarbageCollector
func NewGarbageCollector(rcClient rclientset.Interface, kubeClient kclientset.Interface, rcInformerFactory rinformers.SharedInformerFactory) *GarbageCollector {
	return &GarbageCollector{
		kubeClient: kubeClient,
		rcClient:   rcClient,
		rcLister:   rcInformerFactory.Redisoperator().V1().RedisClusters().Lister(),
		rcSynced:   rcInformerFactory.Redisoperator().V1().RedisClusters().Informer().HasSynced,
	}
}

// InformerSync returns the rediscluster cache informer sync function
func (c *GarbageCollector) InformerSync() cache.InformerSynced {
	return c.rcSynced
}

// CollectRedisClusterGarbage collect the orphaned pods and services. First looking in the rediscluster informer list
// then retrieve from the API and in case NotFound then remove via DeleteCollection primitive
func (c *GarbageCollector) CollectRedisClusterGarbage() error {
	errs := []error{}
	if err := c.collectRedisClusterPods(); err != nil {
		errs = append(errs, err)
	}
	if err := c.collectRedisClusterServices(); err != nil {
		errs = append(errs, err)
	}
	return utilerrors.NewAggregate(errs)
}

// collectRedisClusterPods collect the orphaned pods. First looking in the rediscluster informer list
// then retrieve from the API and in case NotFound then remove via DeleteCollection primitive
func (c *GarbageCollector) collectRedisClusterPods() error {
	glog.V(4).Infof("Collecting garbage pods")
	pods, err := c.kubeClient.CoreV1().Pods(metav1.NamespaceAll).List(metav1.ListOptions{
		LabelSelector: rapi.ClusterNameLabelKey,
	})
	if err != nil {
		return fmt.Errorf("unable to list rediscluster pods to be collected: %v", err)
	}
	errs := []error{}
	collected := make(map[string]struct{})
	for _, pod := range pods.Items {
		redisclusterName, found := pod.Labels[rapi.ClusterNameLabelKey]
		if !found || len(redisclusterName) == 0 {
			errs = append(errs, fmt.Errorf("Unable to find rediscluster name for pod: %s/%s", pod.Namespace, pod.Name))
			continue
		}
		if _, done := collected[path.Join(pod.Namespace, redisclusterName)]; done {
			continue // already collected so skip
		}
		if _, err := c.rcLister.RedisClusters(pod.Namespace).Get(redisclusterName); err == nil || !apierrors.IsNotFound(err) {
			if err != nil {
				errs = append(errs, fmt.Errorf("Unexpected error retrieving rediscluster %s/%s cache: %v", pod.Namespace, redisclusterName, err))
			}
			continue
		}
		// RedisCluster couldn't be find in cache. Trying to get it via APIs.
		if _, err := c.rcClient.RedisoperatorV1().RedisClusters(pod.Namespace).Get(redisclusterName, metav1.GetOptions{}); err != nil {
			if !apierrors.IsNotFound(err) {
				errs = append(errs, fmt.Errorf("Unexpected error retrieving rediscluster %s/%s for pod %s/%s: %v", pod.Namespace, redisclusterName, pod.Namespace, pod.Name, err))
				continue
			}
			// NotFound error: Hence remove all the pods.
			if err := c.kubeClient.CoreV1().Pods(pod.Namespace).DeleteCollection(CascadeDeleteOptions(0), metav1.ListOptions{
				LabelSelector: rapi.ClusterNameLabelKey + "=" + redisclusterName}); err != nil {
				errs = append(errs, fmt.Errorf("Unable to delete Collection of pods for rediscluster %s/%s", pod.Namespace, redisclusterName))
				continue
			}
			collected[path.Join(pod.Namespace, redisclusterName)] = struct{}{} // inserted in the collected map
			glog.Infof("Removed all pods for rediscluster %s/%s", pod.Namespace, redisclusterName)
		}
	}
	return utilerrors.NewAggregate(errs)
}

// collectRedisClusterServices collect the orphaned services. First looking in the rediscluster informer list
// then retrieve from the API and in case NotFound then remove via DeleteCollection primitive
func (c *GarbageCollector) collectRedisClusterServices() error {
	glog.V(4).Infof("Collecting garbage services")
	services, err := c.kubeClient.CoreV1().Services(metav1.NamespaceAll).List(metav1.ListOptions{
		LabelSelector: rapi.ClusterNameLabelKey,
	})
	if err != nil {
		return fmt.Errorf("unable to list rediscluster services to be collected: %v", err)
	}
	errs := []error{}
	collected := make(map[string]struct{})
	for _, service := range services.Items {
		redisclusterName, found := service.Labels[rapi.ClusterNameLabelKey]
		if !found || len(redisclusterName) == 0 {
			errs = append(errs, fmt.Errorf("Unable to find rediscluster name for service: %s/%s", service.Namespace, service.Name))
			continue
		}
		if _, done := collected[path.Join(service.Namespace, redisclusterName)]; done {
			continue // already collected so skip
		}
		if _, err := c.rcLister.RedisClusters(service.Namespace).Get(redisclusterName); err == nil || !apierrors.IsNotFound(err) {
			if err != nil {
				errs = append(errs, fmt.Errorf("Unexpected error retrieving rediscluster %s/%s cache: %v", service.Namespace, redisclusterName, err))
			}
			continue
		}
		// RedisCluster couldn't be find in cache. Trying to get it via APIs.
		if _, err := c.rcClient.RedisoperatorV1().RedisClusters(service.Namespace).Get(redisclusterName, metav1.GetOptions{}); err != nil {
			if !apierrors.IsNotFound(err) {
				errs = append(errs, fmt.Errorf("Unexpected error retrieving rediscluster %s/%s for service %s/%s: %v", service.Namespace, redisclusterName, service.Namespace, service.Name, err))
				continue
			}
			// NotFound error: Hence remove all the pods.
			if err := c.kubeClient.CoreV1().Services(service.Namespace).DeleteCollection(CascadeDeleteOptions(0), metav1.ListOptions{
				LabelSelector: rapi.ClusterNameLabelKey + "=" + redisclusterName}); err != nil {
				errs = append(errs, fmt.Errorf("Unable to delete Collection of services for rediscluster %s/%s", service.Namespace, redisclusterName))
				continue
			}
			collected[path.Join(service.Namespace, redisclusterName)] = struct{}{} // inserted in the collected map
			glog.Infof("Removed all services for rediscluster %s/%s", service.Namespace, redisclusterName)
		}
	}
	return utilerrors.NewAggregate(errs)
}

// CascadeDeleteOptions returns a DeleteOptions with Cascaded set
func CascadeDeleteOptions(gracePeriodSeconds int64) *metav1.DeleteOptions {
	return &metav1.DeleteOptions{
		GracePeriodSeconds: func(t int64) *int64 { return &t }(gracePeriodSeconds),
		PropagationPolicy: func() *metav1.DeletionPropagation {
			background := metav1.DeletePropagationBackground
			return &background
		}(),
	}
}
