package clustering

import (
	"reflect"
	"testing"

	"github.com/amadeusitgroup/redis-operator/pkg/redis"
	"github.com/amadeusitgroup/redis-operator/pkg/redis/fake/admin"
)

func TestAssignSlave(t *testing.T) {
	// TODO currently only test there is no error, more accurate testing is needed
	masterRole := "master"
	slaveRole := "slave"

	redisNode1 := &redis.Node{ID: "1", Role: masterRole, IP: "1.1.1.1", Port: "1234", Slots: append(redis.BuildSlotSlice(10, 20), 0), Pod: newPod("pod1", "vm1")}
	redisNode2 := &redis.Node{ID: "2", Role: masterRole, IP: "1.1.1.2", Port: "1234", Slots: append(redis.BuildSlotSlice(1, 5), redis.BuildSlotSlice(21, 30)...), Pod: newPod("pod2", "vm2")}
	redisNode3 := &redis.Node{ID: "3", Role: slaveRole, MasterReferent: "1", IP: "1.1.1.3", Port: "1234", Slots: []redis.Slot{}, Pod: newPod("pod3", "vm3")}
	redisNode4 := &redis.Node{ID: "4", Role: slaveRole, MasterReferent: "1", IP: "1.1.1.4", Port: "1234", Slots: []redis.Slot{}, Pod: newPod("pod4", "vm4")}
	redisNode5 := &redis.Node{ID: "5", Role: slaveRole, MasterReferent: "1", IP: "1.1.1.5", Port: "1234", Slots: []redis.Slot{}, Pod: newPod("pod5", "vm5")}
	redisNode6 := &redis.Node{ID: "6", Role: masterRole, IP: "1.1.1.6", Port: "1234", Slots: []redis.Slot{}, Pod: newPod("pod6", "vm6")}
	redisNode7 := &redis.Node{ID: "7", Role: masterRole, IP: "1.1.1.7", Port: "1234", Slots: redis.BuildSlotSlice(31, 40), Pod: newPod("pod7", "vm7")}
	redisNode8 := &redis.Node{ID: "8", Role: slaveRole, MasterReferent: "7", IP: "1.1.1.8", Port: "1234", Slots: []redis.Slot{}, Pod: newPod("pod8", "vm8")}
	redisNode9 := &redis.Node{ID: "9", Role: slaveRole, MasterReferent: "7", IP: "1.1.1.9", Port: "1234", Slots: []redis.Slot{}, Pod: newPod("pod9", "vm9")}

	nodes := redis.Nodes{redisNode1, redisNode2, redisNode3, redisNode4, redisNode5, redisNode6, redisNode7, redisNode8, redisNode9}

	c := &redis.Cluster{
		Name:      "clustertest",
		Namespace: "default",
		Nodes: map[string]*redis.Node{
			"1": redisNode1,
			"2": redisNode2,
			"3": redisNode3,
			"4": redisNode4,
			"5": redisNode5,
			"6": redisNode6,
			"7": redisNode7,
			"8": redisNode8,
			"9": redisNode9,
		},
	}

	err := DispatchSlave(c, nodes, 2, admin.NewFakeAdmin([]string{}))
	if err != nil {
		t.Errorf("Unexpected error returned: %v", err)
	}
}

func TestClassifyNodes(t *testing.T) {
	masterRole := "master"
	slaveRole := "slave"
	redisNode1 := &redis.Node{ID: "1", Role: masterRole, IP: "1.1.1.1", Port: "1234", Slots: append(redis.BuildSlotSlice(10, 20), 0), Pod: newPod("pod1", "vm1")}
	redisNode2 := &redis.Node{ID: "2", Role: masterRole, IP: "1.1.1.2", Port: "1234", Slots: append(redis.BuildSlotSlice(1, 5), redis.BuildSlotSlice(21, 30)...), Pod: newPod("pod2", "vm2")}
	redisNode3 := &redis.Node{ID: "3", Role: slaveRole, MasterReferent: "1", IP: "1.1.1.3", Port: "1234", Slots: []redis.Slot{}, Pod: newPod("pod3", "vm3")}
	redisNode4 := &redis.Node{ID: "4", Role: slaveRole, MasterReferent: "1", IP: "1.1.1.4", Port: "1234", Slots: []redis.Slot{}, Pod: newPod("pod4", "vm4")}
	redisNode5 := &redis.Node{ID: "5", Role: slaveRole, MasterReferent: "1", IP: "1.1.1.5", Port: "1234", Slots: []redis.Slot{}, Pod: newPod("pod5", "vm5")}
	redisNode6 := &redis.Node{ID: "6", Role: masterRole, IP: "1.1.1.6", Port: "1234", Slots: []redis.Slot{}, Pod: newPod("pod6", "vm6")}
	redisNode7 := &redis.Node{ID: "7", Role: masterRole, IP: "1.1.1.7", Port: "1234", Slots: redis.BuildSlotSlice(31, 40), Pod: newPod("pod7", "vm7")}
	redisNode8 := &redis.Node{ID: "8", Role: slaveRole, MasterReferent: "7", IP: "1.1.1.8", Port: "1234", Slots: []redis.Slot{}, Pod: newPod("pod8", "vm8")}
	redisNode9 := &redis.Node{ID: "9", Role: slaveRole, MasterReferent: "7", IP: "1.1.1.9", Port: "1234", Slots: []redis.Slot{}, Pod: newPod("pod9", "vm9")}

	nodes := redis.Nodes{redisNode1, redisNode2, redisNode3, redisNode4, redisNode5, redisNode6, redisNode7, redisNode8, redisNode9}

	type args struct {
		nodes redis.Nodes
	}
	tests := []struct {
		name        string
		args        args
		wantMasters redis.Nodes
		wantSlaves  redis.Nodes
		wantNones   redis.Nodes
	}{
		{
			name: "Empty input Nodes slice",
			args: args{
				nodes: redis.Nodes{},
			},
			wantMasters: redis.Nodes{},
			wantSlaves:  redis.Nodes{},
			wantNones:   redis.Nodes{},
		},
		{
			name: "all type of roles",
			args: args{
				nodes: nodes,
			},
			wantMasters: redis.Nodes{redisNode1, redisNode2, redisNode7},
			wantSlaves:  redis.Nodes{redisNode3, redisNode4, redisNode5, redisNode8, redisNode9},
			wantNones:   redis.Nodes{redisNode6},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMasters, gotSlaves, gotNones := ClassifyNodesByRole(tt.args.nodes)
			if !reflect.DeepEqual(gotMasters, tt.wantMasters) {
				t.Errorf("ClassifyNodes() gotMasters = %v, want %v", gotMasters, tt.wantMasters)
			}
			if !reflect.DeepEqual(gotSlaves, tt.wantSlaves) {
				t.Errorf("ClassifyNodes() gotSlaves = %v, want %v", gotSlaves, tt.wantSlaves)
			}
			if !reflect.DeepEqual(gotNones, tt.wantNones) {
				t.Errorf("ClassifyNodes() gotNones = %v, want %v", gotNones, tt.wantNones)
			}
		})
	}
}
