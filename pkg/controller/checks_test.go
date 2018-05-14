package controller

import (
	"encoding/json"
	"reflect"
	"testing"

	kapi "k8s.io/api/core/v1"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rapi "github.com/amadeusitgroup/redis-operator/pkg/api/redis/v1"
	ctrlpod "github.com/amadeusitgroup/redis-operator/pkg/controller/pod"
)

func Test_checkReplicationFactor(t *testing.T) {
	type args struct {
		cluster *rapi.RedisCluster
	}
	tests := []struct {
		name   string
		args   args
		want   map[string][]string
		wantOK bool
	}{
		{
			name:   "On master no slave, as requested",
			want:   map[string][]string{"Master1": {}},
			wantOK: true,
			args: args{
				cluster: &rapi.RedisCluster{
					Spec: rapi.RedisClusterSpec{
						ReplicationFactor: rapi.NewInt32(0),
					},
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{
							MinReplicationFactor: 0,
							MaxReplicationFactor: 0,
							Nodes:                []rapi.RedisClusterNode{{ID: "Master1", Role: rapi.RedisClusterNodeRoleMaster}},
						},
					},
				},
			},
		},
		{
			name:   "On master no slave, missing one slave",
			want:   map[string][]string{"Master1": {}},
			wantOK: false,
			args: args{
				cluster: &rapi.RedisCluster{
					Spec: rapi.RedisClusterSpec{
						ReplicationFactor: rapi.NewInt32(1),
					},
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{
							MinReplicationFactor: 0,
							MaxReplicationFactor: 0,
							Nodes:                []rapi.RedisClusterNode{{ID: "Master1", Role: rapi.RedisClusterNodeRoleMaster}},
						},
					},
				},
			},
		},
		{
			name:   "2 masters, replica=1, missing one slave",
			want:   map[string][]string{"Master1": {"Slave1"}, "Master2": {}},
			wantOK: false,
			args: args{
				cluster: &rapi.RedisCluster{
					Spec: rapi.RedisClusterSpec{
						ReplicationFactor: rapi.NewInt32(1),
					},
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{
							MinReplicationFactor: 0,
							MaxReplicationFactor: 1,
							Nodes: []rapi.RedisClusterNode{
								{ID: "Master1", Role: rapi.RedisClusterNodeRoleMaster},
								{ID: "Master2", Role: rapi.RedisClusterNodeRoleMaster},
								{ID: "Slave1", Role: rapi.RedisClusterNodeRoleSlave, MasterRef: "Master1"},
							},
						},
					},
				},
			},
		},
		{
			name:   "1 master, replica=1, to many slave",
			want:   map[string][]string{"Master1": {"Slave1", "Slave2"}},
			wantOK: false,
			args: args{
				cluster: &rapi.RedisCluster{
					Spec: rapi.RedisClusterSpec{
						ReplicationFactor: rapi.NewInt32(1),
					},
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{
							MinReplicationFactor: 2,
							MaxReplicationFactor: 2,
							Nodes: []rapi.RedisClusterNode{
								{ID: "Master1", Role: rapi.RedisClusterNodeRoleMaster},
								{ID: "Slave1", Role: rapi.RedisClusterNodeRoleSlave, MasterRef: "Master1"},
								{ID: "Slave2", Role: rapi.RedisClusterNodeRoleSlave, MasterRef: "Master1"},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := checkReplicationFactor(tt.args.cluster)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("checkReplicationFactor() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.wantOK {
				t.Errorf("checkReplicationFactor() got1 = %v, want %v", got1, tt.wantOK)
			}
		})
	}
}

func Test_compareStatus(t *testing.T) {
	type args struct {
		old *rapi.RedisClusterClusterStatus
		new *rapi.RedisClusterClusterStatus
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "nothing change (setted)",
			args: args{
				old: &rapi.RedisClusterClusterStatus{},
				new: &rapi.RedisClusterClusterStatus{},
			},
			want: false,
		},
		{
			name: "Cluster status changed",
			args: args{
				old: &rapi.RedisClusterClusterStatus{},
				new: &rapi.RedisClusterClusterStatus{Status: rapi.ClusterStatusKO},
			},
			want: true,
		},
		{
			name: "NumberOfMaster changed",
			args: args{
				old: &rapi.RedisClusterClusterStatus{},
				new: &rapi.RedisClusterClusterStatus{NumberOfMaster: 5},
			},
			want: true,
		},
		{
			name: "MaxReplicationFactor changed",
			args: args{
				old: &rapi.RedisClusterClusterStatus{},
				new: &rapi.RedisClusterClusterStatus{MaxReplicationFactor: 5},
			},
			want: true,
		},
		{
			name: "MinReplicationFactor changed",
			args: args{
				old: &rapi.RedisClusterClusterStatus{},
				new: &rapi.RedisClusterClusterStatus{MinReplicationFactor: 1},
			},
			want: true,
		},
		{
			name: "NbPods changed",
			args: args{
				old: &rapi.RedisClusterClusterStatus{},
				new: &rapi.RedisClusterClusterStatus{NbPods: 1},
			},
			want: true,
		},
		{
			name: "NbPodsReady changed",
			args: args{
				old: &rapi.RedisClusterClusterStatus{},
				new: &rapi.RedisClusterClusterStatus{NbPodsReady: 1},
			},
			want: true,
		},
		{
			name: "NbRedisRunning changed",
			args: args{
				old: &rapi.RedisClusterClusterStatus{},
				new: &rapi.RedisClusterClusterStatus{NbRedisRunning: 5},
			},
			want: true,
		},
		{
			name: "NodesPlacement changed",
			args: args{
				old: &rapi.RedisClusterClusterStatus{},
				new: &rapi.RedisClusterClusterStatus{NodesPlacement: rapi.NodesPlacementInfoOptimal},
			},
			want: true,
		},
		{
			name: "len(Nodes) changed",
			args: args{
				old: &rapi.RedisClusterClusterStatus{},
				new: &rapi.RedisClusterClusterStatus{Nodes: []rapi.RedisClusterNode{{ID: "A"}}},
			},
			want: true,
		},
		{
			name: "Nodes ID changed",
			args: args{
				old: &rapi.RedisClusterClusterStatus{Nodes: []rapi.RedisClusterNode{{ID: "B"}}},
				new: &rapi.RedisClusterClusterStatus{Nodes: []rapi.RedisClusterNode{{ID: "A"}}},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := compareStatus(tt.args.old, tt.args.new); got != tt.want {
				t.Errorf("compareStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_compareNodes(t *testing.T) {
	type args struct {
		nodeA *rapi.RedisClusterNode
		nodeB *rapi.RedisClusterNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "nothing change",
			args: args{
				nodeA: &rapi.RedisClusterNode{},
				nodeB: &rapi.RedisClusterNode{},
			},
			want: false,
		},
		{
			name: "Ip change",
			args: args{
				nodeA: &rapi.RedisClusterNode{IP: "A"},
				nodeB: &rapi.RedisClusterNode{IP: "B"},
			},
			want: true,
		},
		{
			name: "PodName change",
			args: args{
				nodeA: &rapi.RedisClusterNode{PodName: "A"},
				nodeB: &rapi.RedisClusterNode{PodName: "B"},
			},
			want: true,
		},
		{
			name: "MasterRef change",
			args: args{
				nodeA: &rapi.RedisClusterNode{MasterRef: "A"},
				nodeB: &rapi.RedisClusterNode{MasterRef: "B"},
			},
			want: true,
		},
		{
			name: "Port change",
			args: args{
				nodeA: &rapi.RedisClusterNode{Port: "10"},
				nodeB: &rapi.RedisClusterNode{Port: "20"},
			},
			want: true,
		},
		{
			name: "Role change",
			args: args{
				nodeA: &rapi.RedisClusterNode{Role: rapi.RedisClusterNodeRoleMaster},
				nodeB: &rapi.RedisClusterNode{Role: rapi.RedisClusterNodeRoleSlave},
			},
			want: true,
		},
		{
			name: "Slots change",
			args: args{
				nodeA: &rapi.RedisClusterNode{Slots: []string{"1"}},
				nodeB: &rapi.RedisClusterNode{Slots: []string{"1", "2"}},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := compareNodes(tt.args.nodeA, tt.args.nodeB); got != tt.want {
				t.Errorf("compareNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_needMorePods(t *testing.T) {
	type args struct {
		cluster *rapi.RedisCluster
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "no need more pods",
			want: false,
			args: args{
				cluster: &rapi.RedisCluster{
					Spec: rapi.RedisClusterSpec{
						NumberOfMaster:    rapi.NewInt32(3),
						ReplicationFactor: rapi.NewInt32(1),
					},
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{NbPods: 6, NbPodsReady: 6},
					},
				},
			},
		},
		{
			name: "correct number of pod but all pods not ready",
			want: false,
			args: args{
				cluster: &rapi.RedisCluster{
					Spec: rapi.RedisClusterSpec{
						NumberOfMaster:    rapi.NewInt32(3),
						ReplicationFactor: rapi.NewInt32(1),
					},
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{NbPods: 6, NbPodsReady: 4},
					},
				},
			},
		},
		{
			name: "missing pods but all pods not ready",
			want: false,
			args: args{
				cluster: &rapi.RedisCluster{
					Spec: rapi.RedisClusterSpec{
						NumberOfMaster:    rapi.NewInt32(3),
						ReplicationFactor: rapi.NewInt32(1),
					},
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{NbPods: 4, NbPodsReady: 3},
					},
				},
			},
		},
		{
			name: "missing pods and all pods ready",
			want: true,
			args: args{
				cluster: &rapi.RedisCluster{
					Spec: rapi.RedisClusterSpec{
						NumberOfMaster:    rapi.NewInt32(3),
						ReplicationFactor: rapi.NewInt32(1),
					},
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{NbPods: 4, NbPodsReady: 4},
					},
				},
			},
		},
		{
			name: "to many pods",
			want: false,
			args: args{
				cluster: &rapi.RedisCluster{
					Spec: rapi.RedisClusterSpec{
						NumberOfMaster:    rapi.NewInt32(3),
						ReplicationFactor: rapi.NewInt32(1),
					},
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{NbPods: 10, NbPodsReady: 10},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := needMorePods(tt.args.cluster); got != tt.want {
				t.Errorf("needMorePods() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_needLessPods(t *testing.T) {
	type args struct {
		cluster *rapi.RedisCluster
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "no need less pods",
			want: false,
			args: args{
				cluster: &rapi.RedisCluster{
					Spec: rapi.RedisClusterSpec{
						NumberOfMaster:    rapi.NewInt32(3),
						ReplicationFactor: rapi.NewInt32(1),
					},
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{NbPods: 6, NbPodsReady: 6},
					},
				},
			},
		},
		{
			name: "correct number of pod but all pods not ready",
			want: false,
			args: args{
				cluster: &rapi.RedisCluster{
					Spec: rapi.RedisClusterSpec{
						NumberOfMaster:    rapi.NewInt32(3),
						ReplicationFactor: rapi.NewInt32(1),
					},
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{NbPods: 6, NbPodsReady: 4},
					},
				},
			},
		},
		{
			name: "missing pods but all pods not ready",
			want: false,
			args: args{
				cluster: &rapi.RedisCluster{
					Spec: rapi.RedisClusterSpec{
						NumberOfMaster:    rapi.NewInt32(3),
						ReplicationFactor: rapi.NewInt32(1),
					},
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{NbPods: 4, NbPodsReady: 3},
					},
				},
			},
		},
		{
			name: "missing pods and all pods ready",
			want: false,
			args: args{
				cluster: &rapi.RedisCluster{
					Spec: rapi.RedisClusterSpec{
						NumberOfMaster:    rapi.NewInt32(3),
						ReplicationFactor: rapi.NewInt32(1),
					},
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{NbPods: 4, NbPodsReady: 4},
					},
				},
			},
		},
		{
			name: "to many pods",
			want: true,
			args: args{
				cluster: &rapi.RedisCluster{
					Spec: rapi.RedisClusterSpec{
						NumberOfMaster:    rapi.NewInt32(3),
						ReplicationFactor: rapi.NewInt32(1),
					},
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{NbPods: 10, NbPodsReady: 10},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := needLessPods(tt.args.cluster); got != tt.want {
				t.Errorf("needLessPods() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_checkNumberOfMaster(t *testing.T) {
	type args struct {
		cluster *rapi.RedisCluster
	}
	tests := []struct {
		name  string
		args  args
		want  int32
		want1 bool
	}{
		{
			name:  "number of master OK",
			want:  0,
			want1: true,
			args: args{
				cluster: &rapi.RedisCluster{
					Spec: rapi.RedisClusterSpec{
						NumberOfMaster: rapi.NewInt32(3),
					},
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{NumberOfMaster: 3},
					},
				},
			},
		},
		{
			name:  "to many masters",
			want:  3,
			want1: false,
			args: args{
				cluster: &rapi.RedisCluster{
					Spec: rapi.RedisClusterSpec{
						NumberOfMaster: rapi.NewInt32(3),
					},
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{NumberOfMaster: 6},
					},
				},
			},
		},
		{
			name:  "not enough masters",
			want:  -3,
			want1: false,
			args: args{
				cluster: &rapi.RedisCluster{
					Spec: rapi.RedisClusterSpec{
						NumberOfMaster: rapi.NewInt32(6),
					},
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{NumberOfMaster: 3},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := checkNumberOfMaster(tt.args.cluster)
			if got != tt.want {
				t.Errorf("checkNumberOfMaster() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("checkNumberOfMaster() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_checkNoPodsUseless(t *testing.T) {

	master1 := rapi.RedisClusterNode{ID: "master1", Slots: []string{"1"}, Role: rapi.RedisClusterNodeRoleMaster}
	master2 := rapi.RedisClusterNode{ID: "master2", Slots: []string{"2"}, Role: rapi.RedisClusterNodeRoleMaster}
	master3 := rapi.RedisClusterNode{ID: "master3", Slots: []string{"3"}, Role: rapi.RedisClusterNodeRoleMaster}
	master4 := rapi.RedisClusterNode{ID: "master4", Slots: []string{}, Role: rapi.RedisClusterNodeRoleMaster}
	slave1 := rapi.RedisClusterNode{ID: "slave1", Role: rapi.RedisClusterNodeRoleSlave, MasterRef: "master1"}
	slave2 := rapi.RedisClusterNode{ID: "slave2", Role: rapi.RedisClusterNodeRoleSlave, MasterRef: "master2"}
	slave3 := rapi.RedisClusterNode{ID: "slave3", Role: rapi.RedisClusterNodeRoleSlave, MasterRef: "master3"}

	type args struct {
		cluster *rapi.RedisCluster
	}
	tests := []struct {
		name  string
		args  args
		want  []*rapi.RedisClusterNode
		want1 bool
	}{
		{
			name:  "no useless pod",
			want:  []*rapi.RedisClusterNode{},
			want1: true,
			args: args{
				cluster: &rapi.RedisCluster{
					Spec: rapi.RedisClusterSpec{
						NumberOfMaster:    rapi.NewInt32(3),
						ReplicationFactor: rapi.NewInt32(1),
					},
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{
							MinReplicationFactor: 1,
							MaxReplicationFactor: 1,
							NbPods:               6,
							NbPodsReady:          6,
							Nodes: []rapi.RedisClusterNode{
								master1, master2, master3, slave1, slave2, slave3,
							},
						},
					},
				},
			},
		},
		{
			name:  "useless master pod",
			want:  []*rapi.RedisClusterNode{&master4},
			want1: false,
			args: args{
				cluster: &rapi.RedisCluster{
					Spec: rapi.RedisClusterSpec{
						NumberOfMaster:    rapi.NewInt32(3),
						ReplicationFactor: rapi.NewInt32(1),
					},
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{
							MinReplicationFactor: 1,
							MaxReplicationFactor: 1,
							NumberOfMaster:       3,
							NbPods:               7,
							NbPodsReady:          7,
							Nodes: []rapi.RedisClusterNode{
								master1, master2, master3, slave1, slave2, slave3, master4,
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := checkNoPodsUseless(tt.args.cluster)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("checkNoPodsUseless() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("checkNoPodsUseless() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_checkslaveOfSlave(t *testing.T) {
	master1 := rapi.RedisClusterNode{ID: "master1", Slots: []string{"1"}, Role: rapi.RedisClusterNodeRoleMaster}
	master2 := rapi.RedisClusterNode{ID: "master2", Slots: []string{"2"}, Role: rapi.RedisClusterNodeRoleMaster}
	master3 := rapi.RedisClusterNode{ID: "master3", Slots: []string{"3"}, Role: rapi.RedisClusterNodeRoleMaster}
	slave1 := rapi.RedisClusterNode{ID: "slave1", Role: rapi.RedisClusterNodeRoleSlave, MasterRef: "master1"}
	slave2 := rapi.RedisClusterNode{ID: "slave2", Role: rapi.RedisClusterNodeRoleSlave, MasterRef: "master2"}
	slave3 := rapi.RedisClusterNode{ID: "slave3", Role: rapi.RedisClusterNodeRoleSlave, MasterRef: "master3"}
	node4 := rapi.RedisClusterNode{ID: "node4", Role: rapi.RedisClusterNodeRoleSlave, MasterRef: "slave1"}
	node5 := rapi.RedisClusterNode{ID: "node5", Role: rapi.RedisClusterNodeRoleSlave, MasterRef: "slave2"}

	type args struct {
		cluster *rapi.RedisCluster
	}
	tests := []struct {
		name  string
		args  args
		want  map[string][]*rapi.RedisClusterNode
		want1 bool
	}{
		{
			name:  "no slave of slave",
			want:  map[string][]*rapi.RedisClusterNode{},
			want1: true,
			args: args{
				cluster: &rapi.RedisCluster{
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{
							Nodes: []rapi.RedisClusterNode{
								master1, master2, master3, slave1, slave2, slave3,
							},
						},
					},
				},
			},
		},
		{
			name: "2 slaves of slave",
			want: map[string][]*rapi.RedisClusterNode{
				"slave1": {&node4},
				"slave2": {&node5},
			},
			want1: false,
			args: args{
				cluster: &rapi.RedisCluster{
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{
							Nodes: []rapi.RedisClusterNode{
								master1, master2, master3, slave1, slave2, slave3, node4, node5,
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := checkslaveOfSlave(tt.args.cluster)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("checkslaveOfSlave() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("checkslaveOfSlave() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_needClusterOperation(t *testing.T) {
	type args struct {
		cluster *rapi.RedisCluster
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "no operation required",
			want: false,
			args: args{
				cluster: &rapi.RedisCluster{
					Spec: rapi.RedisClusterSpec{
						PodTemplate: &kapi.PodTemplateSpec{
							Spec: kapi.PodSpec{},
						},
						NumberOfMaster:    rapi.NewInt32(1),
						ReplicationFactor: rapi.NewInt32(2),
					},
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{
							NumberOfMaster:       1,
							MinReplicationFactor: 2,
							MaxReplicationFactor: 2,
							NbPods:               3,
							NbPodsReady:          3,
							Nodes: []rapi.RedisClusterNode{
								{ID: "Master1", Role: rapi.RedisClusterNodeRoleMaster},
								{ID: "Slave1", Role: rapi.RedisClusterNodeRoleSlave, MasterRef: "Master1"},
								{ID: "Slave2", Role: rapi.RedisClusterNodeRoleSlave, MasterRef: "Master1"},
							},
						},
					},
				},
			},
		},
		{
			name: "need rollingupdate",
			want: true,
			args: args{
				cluster: &rapi.RedisCluster{
					Spec: rapi.RedisClusterSpec{
						PodTemplate: &kapi.PodTemplateSpec{
							Spec: kapi.PodSpec{
								Containers: []kapi.Container{{Name: "redis", Image: "redis:4.0.6"}},
							},
						},
						NumberOfMaster:    rapi.NewInt32(1),
						ReplicationFactor: rapi.NewInt32(2),
					},
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{
							NumberOfMaster:       1,
							MinReplicationFactor: 2,
							MaxReplicationFactor: 2,
							NbPods:               3,
							NbPodsReady:          3,
							Nodes: []rapi.RedisClusterNode{
								{ID: "Master1", Role: rapi.RedisClusterNodeRoleMaster, Pod: newPodWithContainer("pod1", "vm1", map[string]string{"redis": "redis:4.0.0"})},
								{ID: "Slave1", Role: rapi.RedisClusterNodeRoleSlave, MasterRef: "Master1", Pod: newPodWithContainer("pod2", "vm3", map[string]string{"redis": "redis:4.0.0"})},
								{ID: "Slave2", Role: rapi.RedisClusterNodeRoleSlave, MasterRef: "Master1", Pod: newPodWithContainer("pod3", "vm3", map[string]string{"redis": "redis:4.0.0"})},
							},
						},
					},
				},
			},
		},
		{
			name: "need more pods",
			want: true,
			args: args{
				cluster: &rapi.RedisCluster{
					Spec: rapi.RedisClusterSpec{
						PodTemplate: &kapi.PodTemplateSpec{
							Spec: kapi.PodSpec{},
						},
						NumberOfMaster:    rapi.NewInt32(2),
						ReplicationFactor: rapi.NewInt32(2),
					},
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{
							NumberOfMaster:       1,
							MinReplicationFactor: 2,
							MaxReplicationFactor: 2,
							NbPods:               3,
							NbPodsReady:          3,
							Nodes: []rapi.RedisClusterNode{
								{ID: "Master1", Role: rapi.RedisClusterNodeRoleMaster},
								{ID: "Slave1", Role: rapi.RedisClusterNodeRoleSlave, MasterRef: "Master1"},
								{ID: "Slave2", Role: rapi.RedisClusterNodeRoleSlave, MasterRef: "Master1"},
							},
						},
					},
				},
			},
		},
		{
			name: "need less pods",
			want: true,
			args: args{
				cluster: &rapi.RedisCluster{
					Spec: rapi.RedisClusterSpec{
						PodTemplate:       &kapi.PodTemplateSpec{},
						NumberOfMaster:    rapi.NewInt32(1),
						ReplicationFactor: rapi.NewInt32(2),
					},
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{
							NumberOfMaster:       1,
							MinReplicationFactor: 2,
							MaxReplicationFactor: 2,
							NbPods:               4,
							NbPodsReady:          4,
							Nodes: []rapi.RedisClusterNode{
								{ID: "Master1", Role: rapi.RedisClusterNodeRoleMaster},
								{ID: "Slave1", Role: rapi.RedisClusterNodeRoleSlave, MasterRef: "Master1"},
								{ID: "Slave2", Role: rapi.RedisClusterNodeRoleSlave, MasterRef: "Master1"},
							},
						},
					},
				},
			},
		},
		{
			name: "not enough master",
			want: true,
			args: args{
				cluster: &rapi.RedisCluster{
					Spec: rapi.RedisClusterSpec{
						PodTemplate:       &kapi.PodTemplateSpec{},
						NumberOfMaster:    rapi.NewInt32(1),
						ReplicationFactor: rapi.NewInt32(2),
					},
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{
							NumberOfMaster:       0,
							MinReplicationFactor: 2,
							MaxReplicationFactor: 2,
							NbPods:               3,
							NbPodsReady:          3,
							Nodes: []rapi.RedisClusterNode{
								{ID: "Master1", Role: rapi.RedisClusterNodeRoleMaster},
								{ID: "Slave1", Role: rapi.RedisClusterNodeRoleSlave, MasterRef: "Master1"},
								{ID: "Slave2", Role: rapi.RedisClusterNodeRoleSlave, MasterRef: "Master1"},
							},
						},
					},
				},
			},
		},
		{
			name: "min replica error",
			want: true,
			args: args{
				cluster: &rapi.RedisCluster{
					Spec: rapi.RedisClusterSpec{
						PodTemplate:       &kapi.PodTemplateSpec{},
						NumberOfMaster:    rapi.NewInt32(1),
						ReplicationFactor: rapi.NewInt32(2),
					},
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{
							NumberOfMaster:       1,
							MinReplicationFactor: 1,
							MaxReplicationFactor: 2,
							NbPods:               3,
							NbPodsReady:          3,
							Nodes: []rapi.RedisClusterNode{
								{ID: "Master1", Role: rapi.RedisClusterNodeRoleMaster},
								{ID: "Slave1", Role: rapi.RedisClusterNodeRoleSlave, MasterRef: "Master1"},
								{ID: "Slave2", Role: rapi.RedisClusterNodeRoleSlave, MasterRef: "Master1"},
							},
						},
					},
				},
			},
		},
		{
			name: "max replica error",
			want: true,
			args: args{
				cluster: &rapi.RedisCluster{
					Spec: rapi.RedisClusterSpec{
						PodTemplate:       &kapi.PodTemplateSpec{},
						NumberOfMaster:    rapi.NewInt32(1),
						ReplicationFactor: rapi.NewInt32(2),
					},
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{
							NumberOfMaster:       1,
							MinReplicationFactor: 2,
							MaxReplicationFactor: 3,
							NbPods:               3,
							NbPodsReady:          3,
							Nodes: []rapi.RedisClusterNode{
								{ID: "Master1", Role: rapi.RedisClusterNodeRoleMaster},
								{ID: "Slave1", Role: rapi.RedisClusterNodeRoleSlave, MasterRef: "Master1"},
								{ID: "Slave2", Role: rapi.RedisClusterNodeRoleSlave, MasterRef: "Master1"},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := needClusterOperation(tt.args.cluster); got != tt.want {
				t.Errorf("needClusterOperation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_comparePodsWithPodTemplate(t *testing.T) {
	Node1 := rapi.RedisClusterNode{ID: "slave1", Pod: newPodWithContainer("pod1", "vm1", map[string]string{"redis": "redis:4.0.0"})}
	Node1bis := rapi.RedisClusterNode{ID: "master1", Pod: newPodWithContainer("pod3", "vm3", map[string]string{"redis": "redis:4.0.0"})}

	Node2 := rapi.RedisClusterNode{ID: "master2", Pod: newPodWithContainer("pod2", "vm2", map[string]string{"redis": "redis:4.0.6"})}
	Node2bis := rapi.RedisClusterNode{ID: "master3", Pod: newPodWithContainer("pod4", "vm4", map[string]string{"redis": "redis:4.0.6"})}

	type args struct {
		cluster *rapi.RedisCluster
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "test, no nodes running",
			args: args{
				cluster: &rapi.RedisCluster{
					Spec: rapi.RedisClusterSpec{
						PodTemplate: &kapi.PodTemplateSpec{
							Spec: kapi.PodSpec{
								Containers: []kapi.Container{{Name: "redis", Image: "redis:4.0.0"}},
							},
						},
					},
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{
							Nodes: []rapi.RedisClusterNode{},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "test, two nodes with same version",
			args: args{
				cluster: &rapi.RedisCluster{
					Spec: rapi.RedisClusterSpec{
						PodTemplate: &kapi.PodTemplateSpec{
							Spec: kapi.PodSpec{
								Containers: []kapi.Container{{Name: "redis", Image: "redis:4.0.0"}},
							},
						},
					},
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{
							Nodes: []rapi.RedisClusterNode{
								Node1,
								Node1bis,
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "test, one nodes with different version",
			args: args{
				cluster: &rapi.RedisCluster{
					Spec: rapi.RedisClusterSpec{
						PodTemplate: &kapi.PodTemplateSpec{
							Spec: kapi.PodSpec{
								Containers: []kapi.Container{{Name: "redis", Image: "redis:4.0.6"}},
							},
						},
					},
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{
							Nodes: []rapi.RedisClusterNode{
								Node1,
								Node2,
								Node2bis,
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "test, spec with new container",
			args: args{
				cluster: &rapi.RedisCluster{
					Spec: rapi.RedisClusterSpec{
						PodTemplate: &kapi.PodTemplateSpec{
							Spec: kapi.PodSpec{
								Containers: []kapi.Container{{Name: "redis", Image: "redis:4.0.6"}, {Name: "redis-exporter", Image: "exporter:latest"}},
							},
						},
					},
					Status: rapi.RedisClusterStatus{
						Cluster: rapi.RedisClusterClusterStatus{
							Nodes: []rapi.RedisClusterNode{
								Node2,
								Node2bis,
							},
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := comparePodsWithPodTemplate(tt.args.cluster); got != tt.want {
				t.Errorf("comparePodsWithPodTemplate() = %v, want %v", got, tt.want)

				bwant, _ := json.Marshal(&tt.args.cluster.Spec.PodTemplate.Spec)
				t.Errorf("comparePodsWithPodTemplate() want %s", string(bwant))
				for id, node := range tt.args.cluster.Status.Cluster.Nodes {
					bgot, _ := json.Marshal(&node.Pod.Spec)
					t.Errorf("comparePodsWithPodTemplate() have id:%d %s", id, string(bgot))
				}
			}
		})
	}
}

func newPodWithContainer(name, node string, containersInfos map[string]string) *kapi.Pod {
	var containers []kapi.Container
	for name, image := range containersInfos {
		containers = append(containers, kapi.Container{Name: name, Image: image})
	}

	spec := kapi.PodSpec{
		Containers: containers,
	}

	hash, _ := ctrlpod.GenerateMD5Spec(&spec)

	pod := &kapi.Pod{
		ObjectMeta: kmetav1.ObjectMeta{
			Name:        name,
			Annotations: map[string]string{rapi.PodSpecMD5LabelKey: string(hash)},
		},
		Spec: spec,
	}

	return pod
}

func Test_comparePodSpec(t *testing.T) {
	podSpec1 := kapi.PodSpec{Containers: []kapi.Container{{Name: "redis-node", Image: "redis-node:3.0.3"}}}
	podSpec2 := kapi.PodSpec{Containers: []kapi.Container{{Name: "redis-node", Image: "redis-node:4.0.8"}}}
	podSpec3 := kapi.PodSpec{Containers: []kapi.Container{{Name: "redis-node", Image: "redis-node:3.0.3"}, {Name: "prometheus", Image: "prometheus-exporter:latest"}}}
	hashspec1, _ := ctrlpod.GenerateMD5Spec(&podSpec1)
	hashspec2, _ := ctrlpod.GenerateMD5Spec(&podSpec2)
	hashspec3, _ := ctrlpod.GenerateMD5Spec(&podSpec3)
	type args struct {
		spec string
		pod  *kapi.Pod
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "PodSpecs similar",
			args: args{
				spec: hashspec1,
				pod: &kapi.Pod{
					ObjectMeta: kmetav1.ObjectMeta{
						Annotations: map[string]string{rapi.PodSpecMD5LabelKey: string(hashspec1)},
					},
					Spec: podSpec1,
				},
			},
			want: true,
		},
		{
			name: "PodSpecs not equal",
			args: args{
				spec: hashspec1,
				pod: &kapi.Pod{
					ObjectMeta: kmetav1.ObjectMeta{
						Annotations: map[string]string{rapi.PodSpecMD5LabelKey: string(hashspec2)},
					},
					Spec: podSpec2},
			},
			want: false,
		},
		{
			name: "additional container",
			args: args{
				spec: hashspec1,
				pod: &kapi.Pod{
					ObjectMeta: kmetav1.ObjectMeta{
						Annotations: map[string]string{rapi.PodSpecMD5LabelKey: string(hashspec3)},
					},
					Spec: podSpec3},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := comparePodSpecMD5Hash(tt.args.spec, tt.args.pod); got != tt.want {
				t.Errorf("comparePodSpec() = %v, want %v", got, tt.want)
			}
		})
	}
}
