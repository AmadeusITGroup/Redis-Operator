package sanitycheck

import (
	"reflect"
	"testing"

	kapiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rapi "github.com/amadeusitgroup/redis-operator/pkg/api/redis/v1"
	"github.com/amadeusitgroup/redis-operator/pkg/redis"
	"github.com/amadeusitgroup/redis-operator/pkg/redis/fake/admin"
)

func TestFixUntrustedNodes(t *testing.T) {
	pod1 := newPod("pod1", "node1", "10.0.0.1")
	pod2 := newPod("pod2", "node2", "10.0.0.2")
	pod3 := newPod("pod3", "node3", "10.0.0.3")
	pod4 := newPod("pod3", "node4", "10.0.0.4")
	redis1 := redis.Node{ID: "redis1", Role: "slave", IP: "10.0.0.1", Pod: pod1}
	redis2 := redis.Node{ID: "redis2", Role: "master", IP: "10.0.0.2", Pod: pod2, Slots: []redis.Slot{1}}
	redisUntrusted := redis.Node{ID: "redis3", FailStatus: []string{string(redis.NodeStatusHandshake)}, Role: "master", IP: "10.0.0.3", Pod: pod3, Slots: []redis.Slot{}}
	redis4 := redis.Node{ID: "redis4", Role: "slave", IP: "10.0.0.3", Pod: pod3}

	type args struct {
		adminFunc  func() redis.AdminInterface
		podControl *Fakecontrol
		cluster    *rapi.RedisCluster
		infos      *redis.ClusterInfos
	}
	tests := []struct {
		name           string
		args           args
		want           bool
		wantErr        bool
		wantPodDeleted map[string]bool
	}{
		{
			name: "clean cluster",
			args: args{
				adminFunc: func() redis.AdminInterface {
					nodesAddr := []string{redis1.IPPort(), redis2.IPPort()}
					fakeAdmin := admin.NewFakeAdmin(nodesAddr)

					return fakeAdmin
				},
				podControl: &Fakecontrol{
					pods: []*kapiv1.Pod{pod1, pod2},
				},
				cluster: &rapi.RedisCluster{
					ObjectMeta: metav1.ObjectMeta{Name: "test-cluster", Namespace: "test-ns"},
				},
				infos: &redis.ClusterInfos{
					Infos: map[string]*redis.NodeInfos{
						redis1.ID: {Node: &redis1, Friends: redis.Nodes{&redis2}},
						redis2.ID: {Node: &redis2, Friends: redis.Nodes{&redis1}},
					},
					Status: redis.ClusterInfosConsistent,
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "one node untrusted",
			args: args{
				adminFunc: func() redis.AdminInterface {
					nodesAddr := []string{redis1.IPPort(), redis2.IPPort()}
					fakeAdmin := admin.NewFakeAdmin(nodesAddr)

					return fakeAdmin
				},
				podControl: &Fakecontrol{
					pods: []*kapiv1.Pod{pod1, pod2},
				},
				cluster: &rapi.RedisCluster{
					ObjectMeta: metav1.ObjectMeta{Name: "test-cluster", Namespace: "test-ns"},
				},
				infos: &redis.ClusterInfos{
					Infos: map[string]*redis.NodeInfos{
						redis1.ID: {Node: &redis1, Friends: redis.Nodes{&redis2, &redisUntrusted}},
						redis2.ID: {Node: &redis2, Friends: redis.Nodes{&redis1}},
					},
					Status: redis.ClusterInfosConsistent,
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "pod ip reused",
			args: args{
				adminFunc: func() redis.AdminInterface {
					nodesAddr := []string{redis1.IPPort(), redis2.IPPort()}
					fakeAdmin := admin.NewFakeAdmin(nodesAddr)

					return fakeAdmin
				},
				podControl: &Fakecontrol{
					pods: []*kapiv1.Pod{pod1, pod2, pod3},
				},
				cluster: &rapi.RedisCluster{
					ObjectMeta: metav1.ObjectMeta{Name: "test-cluster", Namespace: "test-ns"},
				},
				infos: &redis.ClusterInfos{
					Infos: map[string]*redis.NodeInfos{
						redis1.ID: {Node: &redis1, Friends: redis.Nodes{&redis2, &redis4, &redisUntrusted}},
						redis2.ID: {Node: &redis2, Friends: redis.Nodes{&redis1, &redis4}},
						redis4.ID: {Node: &redis4, Friends: redis.Nodes{&redis1, &redis2}},
					},
					Status: redis.ClusterInfosConsistent,
				},
			},
			want:    false,
			wantErr: false,
		},

		{
			name: "same ip reused different name",
			args: args{
				adminFunc: func() redis.AdminInterface {
					nodesAddr := []string{redis1.IPPort(), redis2.IPPort()}
					fakeAdmin := admin.NewFakeAdmin(nodesAddr)

					return fakeAdmin
				},
				podControl: newFakecontrol([]*kapiv1.Pod{pod1, pod2, pod3}),
				cluster: &rapi.RedisCluster{
					ObjectMeta: metav1.ObjectMeta{Name: "test-cluster", Namespace: "test-ns"},
				},
				infos: &redis.ClusterInfos{
					Infos: map[string]*redis.NodeInfos{
						redis1.ID: {Node: &redis1, Friends: redis.Nodes{&redis2, &redisUntrusted}},
						redis2.ID: {Node: &redis2, Friends: redis.Nodes{&redis1}},
					},
					Status: redis.ClusterInfosConsistent,
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "same ip reused different name",
			args: args{
				adminFunc: func() redis.AdminInterface {
					nodesAddr := []string{redis1.IPPort(), redis2.IPPort()}
					fakeAdmin := admin.NewFakeAdmin(nodesAddr)

					return fakeAdmin
				},
				podControl: newFakecontrol([]*kapiv1.Pod{pod1, pod2, pod4}),
				cluster: &rapi.RedisCluster{
					ObjectMeta: metav1.ObjectMeta{Name: "test-cluster", Namespace: "test-ns"},
				},
				infos: &redis.ClusterInfos{
					Infos: map[string]*redis.NodeInfos{
						redis1.ID: {Node: &redis1, Friends: redis.Nodes{&redis2, &redisUntrusted}},
						redis2.ID: {Node: &redis2, Friends: redis.Nodes{&redis1}},
					},
					Status: redis.ClusterInfosConsistent,
				},
			},
			want:           true,
			wantErr:        false,
			wantPodDeleted: map[string]bool{"pod3": true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			admin := tt.args.adminFunc()
			got, err := FixUntrustedNodes(admin, tt.args.podControl, tt.args.cluster, tt.args.infos, false)
			if (err != nil) != tt.wantErr {
				t.Errorf("FixUntrustedNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("FixUntrustedNodes() = %v, want %v", got, tt.want)
			}
			if tt.wantPodDeleted != nil {
				if !reflect.DeepEqual(tt.wantPodDeleted, tt.args.podControl.isPodDeleted) {
					t.Errorf("Missing pod deletion current:%v want:%v", tt.args.podControl.isPodDeleted, tt.wantPodDeleted)
					return
				}
			}
		})
	}
}

// Fakecontrol fake control
type Fakecontrol struct {
	pods         []*kapiv1.Pod
	pod          *kapiv1.Pod
	isPodDeleted map[string]bool
}

func newFakecontrol(pods []*kapiv1.Pod) *Fakecontrol {
	return &Fakecontrol{
		pods:         pods,
		isPodDeleted: map[string]bool{},
	}
}

// GetRedisClusterPods return list of Pod attached to a RedisCluster
func (f *Fakecontrol) GetRedisClusterPods(redisCluster *rapi.RedisCluster) ([]*kapiv1.Pod, error) {
	return f.pods, nil
}

// CreatePod used to create a Pod from the RedisCluster pod template
func (f *Fakecontrol) CreatePod(redisCluster *rapi.RedisCluster) (*kapiv1.Pod, error) {
	return f.pod, nil
}

// DeletePod used to delete a pod from its name
func (f *Fakecontrol) DeletePod(redisCluster *rapi.RedisCluster, podName string) error {
	f.isPodDeleted[podName] = true
	return nil
}

// DeletePodNow used to delete a pod from its name
func (f *Fakecontrol) DeletePodNow(redisCluster *rapi.RedisCluster, podName string) error {
	f.isPodDeleted[podName] = true
	return nil
}

func newPod(name, vmName, ip string) *kapiv1.Pod {
	return &kapiv1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name}, Spec: kapiv1.PodSpec{NodeName: vmName}, Status: kapiv1.PodStatus{PodIP: ip}}
}
