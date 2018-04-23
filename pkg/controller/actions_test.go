package controller

import (
	"fmt"
	"reflect"
	"testing"

	rapi "github.com/amadeusitgroup/redis-operator/pkg/api/redis/v1"
	"github.com/amadeusitgroup/redis-operator/pkg/redis"
	"github.com/amadeusitgroup/redis-operator/pkg/redis/fake/admin"
	kapiv1 "k8s.io/api/core/v1"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_searchAvailableSlaveForMasterID(t *testing.T) {
	master1 := redis.Node{ID: "master1", Slots: []redis.Slot{1}, Role: "master"}
	slave1 := redis.Node{ID: "slave1", Slots: []redis.Slot{}, Role: "slave", MasterReferent: "master1"}
	futurslave := redis.Node{ID: "futurslave", Slots: []redis.Slot{}, Role: "master"}

	type args struct {
		nodes          redis.Nodes
		idMaster       string
		nbSlavedNeeded int32
	}
	tests := []struct {
		name    string
		args    args
		want    redis.Nodes
		wantErr bool
	}{
		{
			name: "no slave to search",
			args: args{
				idMaster:       "master1",
				nbSlavedNeeded: 0,
				nodes:          redis.Nodes{&master1},
			},
			want:    redis.Nodes{},
			wantErr: false,
		},
		{
			name: "search one slave",
			args: args{
				idMaster:       "master1",
				nbSlavedNeeded: 1,
				nodes:          redis.Nodes{&master1, &futurslave, &slave1},
			},
			want:    redis.Nodes{&futurslave},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := searchAvailableSlaveForMasterID(tt.args.nodes, tt.args.idMaster, tt.args.nbSlavedNeeded)
			if (err != nil) != tt.wantErr {
				t.Errorf("searchAvailableSlaveForMasterID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("searchAvailableSlaveForMasterID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_selectSlavesToDelete(t *testing.T) {
	master1, redisMaster1 := newRedisMasterNode("1", "pod1", "node1", []string{"1"})
	master2, redisMaster2 := newRedisMasterNode("2", "pod2", "node2", []string{"2"})
	master3, redisMaster3 := newRedisMasterNode("3", "pod3", "node3", []string{"3"})

	slave1, redisSlave1 := newRedisSlaveNode("1", master1.ID, "pod4", "node2")
	slave2, redisSlave2 := newRedisSlaveNode("2", master2.ID, "pod5", "node3")
	slave3, redisSlave3 := newRedisSlaveNode("3", master3.ID, "pod6", "node1")

	slave1bis, redisSlave1bis := newRedisSlaveNode("4", master1.ID, "pod7", "node1")

	type args struct {
		cluster          *rapi.RedisCluster
		nodes            redis.Nodes
		idMaster         string
		slavesID         []string
		nbSlavesToDelete int32
	}
	tests := []struct {
		name    string
		args    args
		want    redis.Nodes
		wantErr bool
	}{
		{
			name:    "one slave to delete",
			want:    redis.Nodes{&redisSlave1bis},
			wantErr: false,
			args: args{
				cluster: &rapi.RedisCluster{
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{
							Nodes: []rapi.RedisClusterNode{
								master1, master2, master3, slave1, slave2, slave3, slave1bis,
							},
						},
					},
				},
				nodes:            redis.Nodes{&redisMaster1, &redisMaster2, &redisMaster3, &redisSlave1, &redisSlave2, &redisSlave3, &redisSlave1bis},
				idMaster:         master1.ID,
				slavesID:         []string{slave1.ID, slave1bis.ID},
				nbSlavesToDelete: 1,
			},
		},
		{
			name: "no slave to delete",
			args: args{
				cluster: &rapi.RedisCluster{
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{
							Nodes: []rapi.RedisClusterNode{
								master1, master2, master3, slave1, slave2, slave3, slave1bis,
							},
						},
					},
				},
				nodes:            redis.Nodes{&redisMaster1, &redisMaster2, &redisMaster3, &redisSlave1, &redisSlave2, &redisSlave3, &redisSlave1bis},
				idMaster:         master1.ID,
				slavesID:         []string{slave1.ID, redisSlave1bis.ID},
				nbSlavesToDelete: 0,
			},
			want:    redis.Nodes{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := selectSlavesToDelete(tt.args.cluster, tt.args.nodes, tt.args.idMaster, tt.args.slavesID, tt.args.nbSlavesToDelete)
			if (err != nil) != tt.wantErr {
				t.Errorf("selectSlavesToDelete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("selectSlavesToDelete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func newRedisSlaveNode(id, masterRef, podName, nodeName string) (rapi.RedisClusterNode, redis.Node) {
	node := newRedisNode(rapi.RedisClusterNodeRoleSlave, id, podName, nodeName, nil)
	node.MasterRef = masterRef
	role := "Slave"
	redisNode := redis.Node{ID: fmt.Sprintf("%s%s", role, id), MasterReferent: masterRef, Role: role, Pod: node.Pod}
	return node, redisNode
}

func newRedisMasterNode(id, podName, nodeName string, slots []string) (rapi.RedisClusterNode, redis.Node) {
	role := "master"
	slotsInt := []redis.Slot{}
	for _, s := range slots {
		i, _ := redis.DecodeSlot(s)
		slotsInt = append(slotsInt, i)
	}
	node := newRedisNode(rapi.RedisClusterNodeRoleMaster, id, podName, nodeName, slots)
	redisNode := redis.Node{ID: fmt.Sprintf("%s%s", role, id), Slots: slotsInt, Role: role, Pod: node.Pod}
	return node, redisNode
}

func newRedisNode(role rapi.RedisClusterNodeRole, id, podName, nodeName string, slots []string) rapi.RedisClusterNode {
	pod := newPod(podName, nodeName)

	return rapi.RedisClusterNode{ID: fmt.Sprintf("%s%s", string(role), id), Slots: slots, Role: role, Pod: pod}
}

func newPod(name, node string) *kapiv1.Pod {
	return &kapiv1.Pod{
		ObjectMeta: kmetav1.ObjectMeta{Name: name},
		Spec: kapiv1.PodSpec{
			NodeName: node,
		},
	}
}

func Test_newRedisCluster(t *testing.T) {
	redis1 := redis.Node{ID: "redis1", Role: "slave", IP: "10.0.0.1", Pod: newPod("pod1", "node1")}
	redis2 := redis.Node{ID: "redis2", Role: "master", IP: "10.0.0.2", Pod: newPod("pod2", "node2"), Slots: []redis.Slot{1}}

	nodesAddr := []string{redis1.IPPort(), redis2.IPPort()}
	fakeAdmin := admin.NewFakeAdmin(nodesAddr)
	fakeAdmin.GetClusterInfosRet = admin.ClusterInfosRetType{
		ClusterInfos: &redis.ClusterInfos{
			Infos: map[string]*redis.NodeInfos{
				redis1.ID: {Node: &redis1, Friends: redis.Nodes{&redis2}},
				redis2.ID: {Node: &redis2, Friends: redis.Nodes{&redis1}},
			},
			Status: redis.ClusterInfosConsistent,
		},
		Err: nil,
	}

	type args struct {
		admin   redis.AdminInterface
		cluster *rapi.RedisCluster
	}
	tests := []struct {
		name    string
		args    args
		want    *redis.Cluster
		want1   redis.Nodes
		wantErr bool
	}{
		{
			name: "create redis cluster",
			args: args{
				admin: fakeAdmin,
				cluster: &rapi.RedisCluster{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:      "myCluster",
						Namespace: "myNamespace",
					},
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{
							Nodes: []rapi.RedisClusterNode{},
						},
					},
				},
			},
			want: &redis.Cluster{
				Name:      "myCluster",
				Namespace: "myNamespace",
				Nodes: map[string]*redis.Node{
					redis1.ID: &redis1,
					redis2.ID: &redis2,
				},
			},
			want1:   redis.Nodes{&redis1, &redis2},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := newRedisCluster(tt.args.admin, tt.args.cluster)
			if (err != nil) != tt.wantErr {
				t.Errorf("newRedisCluster() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newRedisCluster() got = %v, want %v", got, tt.want)
			}
			got1.SortNodes()
			tt.want1.SortNodes()
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("newRedisCluster() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestController_applyConfiguration(t *testing.T) {
	redis1 := redis.Node{ID: "redis1", Role: "slave", IP: "10.0.0.1", Pod: newPod("pod1", "node1")}
	redis2 := redis.Node{ID: "redis2", Role: "master", IP: "10.0.0.2", Pod: newPod("pod2", "node2"), Slots: []redis.Slot{1}}
	redis3 := redis.Node{ID: "redis3", Role: "master", IP: "10.0.0.3", Pod: newPod("pod3", "node3"), Slots: []redis.Slot{}}
	redis4 := redis.Node{ID: "redis4", Role: "master", IP: "10.0.0.4", Pod: newPod("pod4", "node4"), Slots: []redis.Slot{}}

	nodesAddr := []string{redis1.IPPort(), redis2.IPPort()}

	type args struct {
		cluster             *rapi.RedisCluster
		updateFakeAdminFunc func(adm *admin.Admin)
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "nothing to apply",
			args: args{
				updateFakeAdminFunc: func(adm *admin.Admin) {
					adm.GetClusterInfosRet = admin.ClusterInfosRetType{
						ClusterInfos: &redis.ClusterInfos{
							Infos: map[string]*redis.NodeInfos{
								redis1.ID: {Node: &redis1, Friends: redis.Nodes{&redis2}},
								redis2.ID: {Node: &redis2, Friends: redis.Nodes{&redis1}},
							},
							Status: redis.ClusterInfosConsistent,
						},
						Err: nil,
					}
				},
				cluster: &rapi.RedisCluster{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:      "myCluster",
						Namespace: "myNamespace",
					},
					Spec: rapi.RedisClusterSpec{
						PodTemplate:       &kapiv1.PodTemplateSpec{},
						NumberOfMaster:    rapi.NewInt32(1),
						ReplicationFactor: rapi.NewInt32(1),
					},
					Status: rapi.RedisClusterStatus{
						Conditions: []rapi.RedisClusterCondition{},
						Cluster: rapi.RedisClusterClusterStatus{
							NumberOfMaster:       1,
							MinReplicationFactor: 1,
							MaxReplicationFactor: 1,
							NbPods:               2,
							NbPodsReady:          2,
							NbRedisRunning:       2,
							Nodes:                []rapi.RedisClusterNode{},
						},
					},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "new master and slave to apply",
			args: args{
				updateFakeAdminFunc: func(adm *admin.Admin) {
					adm.GetClusterInfosRet = admin.ClusterInfosRetType{
						ClusterInfos: &redis.ClusterInfos{
							Infos: map[string]*redis.NodeInfos{
								redis1.ID: {Node: &redis1, Friends: redis.Nodes{&redis2, &redis3, &redis4}},
								redis2.ID: {Node: &redis2, Friends: redis.Nodes{&redis1, &redis3, &redis4}},
								redis3.ID: {Node: &redis3, Friends: redis.Nodes{&redis2, &redis1, &redis4}},
								redis4.ID: {Node: &redis4, Friends: redis.Nodes{&redis1, &redis2, &redis3}},
							},
							Status: redis.ClusterInfosConsistent,
						},
						Err: nil,
					}
				},
				cluster: &rapi.RedisCluster{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:      "myCluster",
						Namespace: "myNamespace",
					},
					Spec: rapi.RedisClusterSpec{
						PodTemplate:       &kapiv1.PodTemplateSpec{},
						NumberOfMaster:    rapi.NewInt32(2),
						ReplicationFactor: rapi.NewInt32(1),
					},
					Status: rapi.RedisClusterStatus{
						Conditions: []rapi.RedisClusterCondition{},
						Cluster: rapi.RedisClusterClusterStatus{
							NumberOfMaster:       1,
							MinReplicationFactor: 1,
							MaxReplicationFactor: 1,
							NbPods:               4,
							NbPodsReady:          4,
							NbRedisRunning:       4,
							Nodes:                []rapi.RedisClusterNode{},
						},
					},
				},
			},
			want:    true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{
				updateHandler: func(rc *rapi.RedisCluster) (*rapi.RedisCluster, error) { return rc, nil },
			}
			fakeAdmin := admin.NewFakeAdmin(nodesAddr)
			tt.args.updateFakeAdminFunc(fakeAdmin)

			got, err := c.applyConfiguration(fakeAdmin, tt.args.cluster)
			if (err != nil) != tt.wantErr {
				t.Errorf("Controller.applyConfiguration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Controller.applyConfiguration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getOldNodesToRemove(t *testing.T) {
	redis1 := &redis.Node{ID: "redis1", Role: "slave", MasterReferent: "redis2", IP: "10.0.0.1", Pod: newPod("pod1", "node1")}
	redis2 := &redis.Node{ID: "redis2", Role: "master", IP: "10.0.0.2", Pod: newPod("pod2", "node2"), Slots: []redis.Slot{1}}
	redis3 := &redis.Node{ID: "redis3", Role: "master", IP: "10.0.0.3", Pod: newPod("pod3", "node3"), Slots: []redis.Slot{2}}
	redis4 := &redis.Node{ID: "redis4", Role: "master", IP: "10.0.0.4", Pod: newPod("pod4", "node4"), Slots: []redis.Slot{3}}
	redis5 := &redis.Node{ID: "redis5", Role: "slave", MasterReferent: "redis3", IP: "10.0.0.5", Pod: newPod("pod5", "node5")}
	redis6 := &redis.Node{ID: "redis6", Role: "slave", MasterReferent: "redis4", IP: "10.0.0.6", Pod: newPod("pod6", "node6")}

	type args struct {
		curMasters redis.Nodes
		newMasters redis.Nodes
		nodes      redis.Nodes
	}
	tests := []struct {
		name               string
		args               args
		wantRemovedMasters redis.Nodes
		wantRemoveSlaves   redis.Nodes
	}{
		{
			name: "basic test",
			args: args{
				curMasters: redis.Nodes{redis2, redis3, redis4},
				newMasters: redis.Nodes{redis2, redis3},
				nodes:      redis.Nodes{redis1, redis2, redis3, redis4, redis5, redis6},
			},
			wantRemovedMasters: redis.Nodes{redis4},
			wantRemoveSlaves:   redis.Nodes{redis6},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRemovedMasters, gotRemoveSlaves := getOldNodesToRemove(tt.args.curMasters, tt.args.newMasters, tt.args.nodes)
			if !reflect.DeepEqual(gotRemovedMasters, tt.wantRemovedMasters) {
				t.Errorf("getOldNodesToRemove() gotRemovedMasters = %v, want %v", gotRemovedMasters, tt.wantRemovedMasters)
			}
			if !reflect.DeepEqual(gotRemoveSlaves, tt.wantRemoveSlaves) {
				t.Errorf("getOldNodesToRemove() gotRemoveSlaves = %v, want %v", gotRemoveSlaves, tt.wantRemoveSlaves)
			}
		})
	}
}
