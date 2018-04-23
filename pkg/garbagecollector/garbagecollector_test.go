package garbagecollector

import (
	"fmt"
	"net/http"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"

	redis "github.com/amadeusitgroup/redis-operator/pkg/api/redis"
	v1 "github.com/amadeusitgroup/redis-operator/pkg/api/redis/v1"
	rfake "github.com/amadeusitgroup/redis-operator/pkg/client/clientset/versioned/fake"
	rclister "github.com/amadeusitgroup/redis-operator/pkg/client/listers/redis/v1"
)

type FakeRedisClusterLister struct {
	rediscluster []*v1.RedisCluster
}

func (f FakeRedisClusterLister) List(labels.Selector) (ret []*v1.RedisCluster, err error) {
	return f.rediscluster, nil
}

func (f FakeRedisClusterLister) RedisClusters(namespace string) rclister.RedisClusterNamespaceLister {
	return &FakeRedisClusterNamespaceLister{
		namespace: namespace,
		Lister:    f,
	}
}

type FakeRedisClusterNamespaceLister struct {
	namespace string
	Lister    FakeRedisClusterLister
}

func (f FakeRedisClusterNamespaceLister) List(labels.Selector) (ret []*v1.RedisCluster, err error) {
	return []*v1.RedisCluster{}, nil
}

func (f FakeRedisClusterNamespaceLister) Get(name string) (*v1.RedisCluster, error) {
	for i := range f.Lister.rediscluster {
		if f.Lister.rediscluster[i].Name == name {
			return f.Lister.rediscluster[i], nil
		}
	}
	return nil, newRedisClusterNotFoundError(name)
}

func TestGarbageCollector_CollectRedisClusterJobs(t *testing.T) {
	tests := map[string]struct {
		TweakGarbageCollector func(*GarbageCollector) *GarbageCollector
		errorMessage          string
	}{
		"nominal case": {
			// getting 4 pods:
			//   2 to be removed 2 (no rediscluster)
			//   2 to be preserved (found rediscluster)
			TweakGarbageCollector: func(gc *GarbageCollector) *GarbageCollector {
				fakeClient := &fake.Clientset{}
				fakeClient.AddReactor("list", "pods", func(action core.Action) (bool, runtime.Object, error) {
					pods := &corev1.PodList{
						Items: []corev1.Pod{
							createPodWithLabels("testpod1", map[string]string{
								v1.ClusterNameLabelKey: "rediscluster1",
							}),
							createPodWithLabels("testpod2", map[string]string{
								v1.ClusterNameLabelKey: "rediscluster1",
							}),
							createPodWithLabels("testpod3", map[string]string{
								v1.ClusterNameLabelKey: "rediscluster2",
							}),
							createPodWithLabels("testpod4", map[string]string{
								v1.ClusterNameLabelKey: "rediscluster2",
							}),
						},
					}
					return true, pods, nil
				})
				fakeClient.AddReactor("list", "services", func(action core.Action) (bool, runtime.Object, error) {
					svcs := &corev1.ServiceList{
						Items: []corev1.Service{
							createServiceWithLabels("testsvc1", map[string]string{
								v1.ClusterNameLabelKey: "rediscluster1",
							}),
							createServiceWithLabels("testsvc2", map[string]string{
								v1.ClusterNameLabelKey: "rediscluster2",
							}),
						},
					}
					return true, svcs, nil
				})
				gc.kubeClient = fakeClient
				gc.rcLister = &FakeRedisClusterLister{
					rediscluster: []*v1.RedisCluster{
						createRedisCluster("rediscluster2"),
					},
				}
				return gc
			},
			errorMessage: "",
		},
		"no pods and services": {
			errorMessage: "",
		},
		"error getting pods and services": {
			TweakGarbageCollector: func(gc *GarbageCollector) *GarbageCollector {
				fakeClient := &fake.Clientset{}
				fakeClient.AddReactor("list", "pods", func(action core.Action) (bool, runtime.Object, error) {
					return true, nil, apierrors.NewNotFound(action.GetResource().GroupResource(), action.GetResource().Resource)
				})
				fakeClient.AddReactor("list", "services", func(action core.Action) (bool, runtime.Object, error) {
					return true, nil, apierrors.NewNotFound(action.GetResource().GroupResource(), action.GetResource().Resource)
				})
				gc.kubeClient = fakeClient
				return gc
			},
			errorMessage: "[unable to list rediscluster pods to be collected: pods \"pods\" not found, unable to list rediscluster services to be collected: services \"services\" not found]",
		},
		"error getting pod with label without value": {
			TweakGarbageCollector: func(gc *GarbageCollector) *GarbageCollector {
				fakeClient := &fake.Clientset{}
				fakeClient.AddReactor("list", "pods", func(action core.Action) (bool, runtime.Object, error) {
					pod := createPodWithLabels("testpod", map[string]string{
						v1.ClusterNameLabelKey: "",
					})
					var pods runtime.Object
					pods = &corev1.PodList{
						Items: []corev1.Pod{
							pod,
						}}
					return true, pods, nil
				})
				gc.kubeClient = fakeClient
				return gc
			},
			errorMessage: "Unable to find rediscluster name for pod: /testpod",
		},
		"error getting service with label without value": {
			TweakGarbageCollector: func(gc *GarbageCollector) *GarbageCollector {
				fakeClient := &fake.Clientset{}
				fakeClient.AddReactor("list", "services", func(action core.Action) (bool, runtime.Object, error) {
					svc := createServiceWithLabels("testservice", map[string]string{
						v1.ClusterNameLabelKey: "",
					})
					var svcs runtime.Object
					svcs = &corev1.ServiceList{
						Items: []corev1.Service{
							svc,
						}}
					return true, svcs, nil
				})
				gc.kubeClient = fakeClient
				return gc
			},
			errorMessage: "Unable to find rediscluster name for service: /testservice",
		},
		"no rediscluster found for pod": {
			TweakGarbageCollector: func(gc *GarbageCollector) *GarbageCollector {
				fakeClient := &fake.Clientset{}
				fakeClient.AddReactor("list", "pods", func(action core.Action) (bool, runtime.Object, error) {
					pod := createPodWithLabels("testpod", map[string]string{
						v1.ClusterNameLabelKey: "foo",
					})
					var pods runtime.Object
					pods = &corev1.PodList{
						Items: []corev1.Pod{
							pod,
						}}
					return true, pods, nil
				})
				gc.kubeClient = fakeClient
				return gc
			},
			errorMessage: "",
		},
	}
	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			GC := &GarbageCollector{
				kubeClient: fake.NewSimpleClientset(),
				rcClient:   rfake.NewSimpleClientset(),
				rcLister:   FakeRedisClusterLister{},
				rcSynced:   func() bool { return true },
			}
			if tt.TweakGarbageCollector != nil {
				GC = tt.TweakGarbageCollector(GC)
			}
			err := GC.CollectRedisClusterGarbage()
			errorMessage := ""
			if err != nil {
				errorMessage = err.Error()
			}
			if tt.errorMessage != errorMessage {
				t.Errorf("%q\nExpected error: `%s`\nBut got       : `%s`\n", tn, tt.errorMessage, errorMessage)
			}
		})
	}
}

func createPodWithLabels(name string, labels map[string]string) corev1.Pod {
	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Spec: corev1.PodSpec{},
	}
}

func createServiceWithLabels(name string, labels map[string]string) corev1.Service {
	return corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Spec: corev1.ServiceSpec{},
	}
}

func createRedisCluster(name string) *v1.RedisCluster {
	return &v1.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func newRedisClusterNotFoundError(name string) *apierrors.StatusError {
	return &apierrors.StatusError{
		ErrStatus: metav1.Status{
			Status: metav1.StatusFailure,
			Code:   http.StatusNotFound,
			Reason: metav1.StatusReasonNotFound,
			Details: &metav1.StatusDetails{
				Group: redis.GroupName,
				Kind:  v1.ResourcePlural + "." + redis.GroupName,
				Name:  name,
			},
			Message: fmt.Sprintf("%s %q not found", v1.ResourcePlural+"."+redis.GroupName, name),
		}}
}
