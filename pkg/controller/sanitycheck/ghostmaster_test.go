package sanitycheck

import (
	"testing"

	kapiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rapi "github.com/amadeusitgroup/redis-operator/pkg/api/redis/v1"
	"github.com/amadeusitgroup/redis-operator/pkg/controller/pod"
	"github.com/amadeusitgroup/redis-operator/pkg/redis"
	"github.com/amadeusitgroup/redis-operator/pkg/redis/fake/admin"
)

func TestFixGhostMasterNodes(t *testing.T) {
	pod1 := newPod("pod1", "node1", "10.0.0.1")
	pod2 := newPod("pod2", "node2", "10.0.0.2")
	pod3 := newPod("pod3", "node3", "10.0.0.3")
	redis1 := redis.Node{ID: "redis1", Role: "slave", IP: "10.0.0.1", Pod: pod1}
	redis2 := redis.Node{ID: "redis2", Role: "master", IP: "10.0.0.2", Pod: pod2, Slots: []redis.Slot{1}}
	redisGhostMaster := redis.Node{ID: "redis3", Role: "master", IP: "10.0.0.3", Pod: pod3, Slots: []redis.Slot{}}

	type args struct {
		adminFunc  func() redis.AdminInterface
		podControl pod.RedisClusterControlInteface
		cluster    *rapi.RedisCluster
		info       *redis.ClusterInfos
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{

		{
			name: "no gost",
			args: args{
				adminFunc: func() redis.AdminInterface {
					nodesAddr := []string{redis1.IPPort(), redis2.IPPort()}
					fakeAdmin := admin.NewFakeAdmin(nodesAddr)

					return fakeAdmin
				},
				podControl: newFakecontrol([]*kapiv1.Pod{pod1, pod2}),
				cluster: &rapi.RedisCluster{
					ObjectMeta: metav1.ObjectMeta{Name: "test-cluster", Namespace: "test-ns"},
				},
				info: &redis.ClusterInfos{
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
			name: "gost redis3",
			args: args{
				adminFunc: func() redis.AdminInterface {
					nodesAddr := []string{redis1.IPPort(), redis2.IPPort()}
					fakeAdmin := admin.NewFakeAdmin(nodesAddr)

					return fakeAdmin
				},
				podControl: newFakecontrol([]*kapiv1.Pod{pod1, pod2}),
				cluster: &rapi.RedisCluster{
					ObjectMeta: metav1.ObjectMeta{Name: "test-cluster", Namespace: "test-ns"},
				},
				info: &redis.ClusterInfos{
					Infos: map[string]*redis.NodeInfos{
						redis1.ID: {Node: &redis1, Friends: redis.Nodes{&redis2, &redisGhostMaster}},
						redis2.ID: {Node: &redis2, Friends: redis.Nodes{&redis1, &redisGhostMaster}},
					},
					Status: redis.ClusterInfosConsistent,
				},
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			admin := tt.args.adminFunc()
			got, err := FixGhostMasterNodes(admin, tt.args.podControl, tt.args.cluster, tt.args.info)
			if (err != nil) != tt.wantErr {
				t.Errorf("FixGhostMasterNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("FixGhostMasterNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}
