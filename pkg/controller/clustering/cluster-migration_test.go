package clustering

import (
	"reflect"
	"sort"
	"testing"

	"github.com/amadeusitgroup/redis-operator/pkg/redis"
	"github.com/amadeusitgroup/redis-operator/pkg/redis/fake/admin"
)

func TestDispatchSlotToMaster(t *testing.T) {
	simpleAdmin := admin.NewFakeAdmin([]string{})
	pod1 := newPod("pod1", "vm1")
	pod2 := newPod("pod2", "vm2")
	pod3 := newPod("pod3", "vm3")
	pod4 := newPod("pod4", "vm4")

	masterRole := "master"

	redisNode1 := &redis.Node{ID: "1", Role: masterRole, IP: "1.1.1.1", Port: "1234", Slots: redis.BuildSlotSlice(0, simpleAdmin.GetHashMaxSlot()), Pod: pod1}
	redisNode2 := &redis.Node{ID: "2", Role: masterRole, IP: "1.1.1.2", Port: "1234", Slots: []redis.Slot{}, Pod: pod2}
	redisNode3 := &redis.Node{ID: "3", Role: masterRole, IP: "1.1.1.3", Port: "1234", Slots: []redis.Slot{}, Pod: pod3}
	redisNode4 := &redis.Node{ID: "4", Role: masterRole, IP: "1.1.1.4", Port: "1234", Slots: []redis.Slot{}, Pod: pod4}

	testCases := []struct {
		cluster   *redis.Cluster
		nodes     redis.Nodes
		nbMasters int32
		fakeAdmin redis.AdminInterface
		err       bool
	}{
		// append force copy, because DispatchSlotToMaster updates the slic
		{
			cluster: &redis.Cluster{
				Name:      "clustertest",
				Namespace: "default",
				Nodes: map[string]*redis.Node{
					"1": redisNode1,
					"2": redisNode2,
					"3": redisNode3,
					"4": redisNode4,
				},
			},
			nodes: redis.Nodes{
				redisNode1,
				redisNode2,
				redisNode3,
				redisNode4,
			},
			nbMasters: 6, fakeAdmin: simpleAdmin, err: true,
		},
		// not enough master
		{
			cluster: &redis.Cluster{
				Name:      "clustertest",
				Namespace: "default",
				Nodes: map[string]*redis.Node{
					"1": redisNode1,
					"2": redisNode2,
					"3": redisNode3,
					"4": redisNode4,
				},
			},
			nodes: redis.Nodes{
				redisNode1,
				redisNode2,
				redisNode3,
				redisNode4,
			},
			nbMasters: 2, fakeAdmin: simpleAdmin, err: false,
		}, // initial config

		{
			cluster: &redis.Cluster{
				Name:      "clustertest",
				Namespace: "default",
				Nodes: map[string]*redis.Node{
					"1": redisNode1,
				},
			},
			nodes: redis.Nodes{
				redisNode1,
			},
			nbMasters: 1, fakeAdmin: simpleAdmin, err: false,
		}, // only one node

		{
			cluster: &redis.Cluster{
				Name:      "clustertest",
				Namespace: "default",
				Nodes: map[string]*redis.Node{
					"2": redisNode2,
				},
			},
			nodes: redis.Nodes{
				redisNode2,
			},
			nbMasters: 1, fakeAdmin: simpleAdmin, err: false,
		}, // only one node with no slots
		{
			cluster: &redis.Cluster{
				Name:      "clustertest",
				Namespace: "default",
			},
			nodes: redis.Nodes{}, nbMasters: 0, fakeAdmin: simpleAdmin, err: false,
		}, // empty

	}

	for i, tc := range testCases {
		_, _, _, err := DispatchMasters(tc.cluster, tc.nodes, tc.nbMasters, tc.fakeAdmin)
		if (err != nil) != tc.err {
			t.Errorf("[case: %d] Unexpected error status, expected error to be %t, got '%v'", i, tc.err, err)
		}
	}
}

func Test_retrieveLostSlots(t *testing.T) {
	redis1 := &redis.Node{ID: "redis1"}
	redis2 := &redis.Node{ID: "redis2"}
	redis3MissingSlots := &redis.Node{ID: "redis3"}
	nbslots := 16384
	for i := 0; i < nbslots; i++ {
		if i < (nbslots / 2) {
			redis1.Slots = append(redis1.Slots, redis.Slot(i))
		} else {
			redis2.Slots = append(redis2.Slots, redis.Slot(i))
			if i != 16383 {
				redis3MissingSlots.Slots = append(redis3MissingSlots.Slots, redis.Slot(i))
			}
		}
	}
	type args struct {
		oldMasterNodes redis.Nodes
		nbSlots        int
	}
	tests := []struct {
		name string
		args args
		want []redis.Slot
	}{
		{
			name: "no losted slots",
			args: args{
				oldMasterNodes: redis.Nodes{
					redis1,
					redis2,
				},
				nbSlots: nbslots,
			},
			want: []redis.Slot{},
		},
		{
			name: "one losted slot",
			args: args{
				oldMasterNodes: redis.Nodes{
					redis1,
					redis3MissingSlots,
				},
				nbSlots: nbslots,
			},
			want: []redis.Slot{16383},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := retrieveLostSlots(tt.args.oldMasterNodes, tt.args.nbSlots); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("retrieveLostSlots() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_buildSlotByNodeFromAvailableSlots(t *testing.T) {
	redis1 := &redis.Node{ID: "redis1", Slots: []redis.Slot{1, 2, 3, 4, 5, 6, 7, 8, 9}}
	redis2 := &redis.Node{ID: "redis2", Slots: []redis.Slot{}}
	redis3 := &redis.Node{ID: "redis3", Slots: []redis.Slot{}}

	type args struct {
		newMasterNodes      redis.Nodes
		nbSlotByNode        int
		slotToMigrateByNode map[string][]redis.Slot
	}
	tests := []struct {
		name string
		args args
		// we don't know on which nodes the slots will be assign (due to map ordering).
		// the only think that we know is the repartition between nodes (nb slots by node)
		want [][]redis.Slot
	}{
		{
			name: "no nodes",
			args: args{
				newMasterNodes:      redis.Nodes{},
				nbSlotByNode:        0,
				slotToMigrateByNode: map[string][]redis.Slot{},
			},
			want: [][]redis.Slot{},
		},
		{
			name: "no slot to dispatch",
			args: args{
				newMasterNodes: redis.Nodes{
					&redis.Node{ID: "redis1", Slots: []redis.Slot{1, 2, 3}},
					&redis.Node{ID: "redis2", Slots: []redis.Slot{4, 5, 6}},
					&redis.Node{ID: "redis3", Slots: []redis.Slot{7, 8, 9}},
				},
				nbSlotByNode: 3,
				slotToMigrateByNode: map[string][]redis.Slot{
					redis1.ID: {},
					redis2.ID: {},
					redis3.ID: {},
				},
			},
			want: [][]redis.Slot{},
		},
		{
			name: "scale from 1 node to 3 nodes",
			args: args{
				newMasterNodes: redis.Nodes{redis1, redis2, redis3},
				nbSlotByNode:   3,
				slotToMigrateByNode: map[string][]redis.Slot{
					redis1.ID: {4, 5, 6, 7, 8, 9},
					redis2.ID: {},
					redis3.ID: {},
				},
			},
			want: [][]redis.Slot{
				// we don't know on which nodes the slots will be assign (due to map ordering).
				// the only think that we know is the repartition between nodes (nb slots by node)
				{4, 5, 6},
				{7, 8, 9},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildSlotByNodeFromAvailableSlots(tt.args.newMasterNodes, tt.args.nbSlotByNode, tt.args.slotToMigrateByNode)
			gotslice := [][]redis.Slot{}
			for _, slots := range got {
				gotslice = append(gotslice, slots)
			}
			sort.Slice(gotslice, func(a, b int) bool {
				i := gotslice[a]
				j := gotslice[b]
				if len(i) == 0 && len(j) == 0 {
					return false
				}
				if len(i) < len(j) {
					return true
				} else if len(j) >= 1 {
					if i[0] < j[0] {
						return true
					}
				}
				return false
			})

			if !reflect.DeepEqual(gotslice, tt.want) {
				t.Errorf("buildSlotByNodeFromAvailableSlots() = %v, want %v", gotslice, tt.want)
			}
		})
	}
}

func Test_buildSlotsByNode(t *testing.T) {
	redis1 := &redis.Node{ID: "redis1", Slots: []redis.Slot{0, 1, 2, 3, 4, 5, 6, 7, 8}}
	redis2 := &redis.Node{ID: "redis2", Slots: []redis.Slot{}}
	redis3 := &redis.Node{ID: "redis3", Slots: []redis.Slot{}}

	redis4 := &redis.Node{ID: "redis4", Slots: []redis.Slot{0, 1, 2, 3, 4}}
	redis5 := &redis.Node{ID: "redis5", Slots: []redis.Slot{5, 6, 7, 8}}

	type args struct {
		newMasterNodes redis.Nodes
		oldMasterNodes redis.Nodes
		allMasterNodes redis.Nodes
		nbSlots        int
	}
	tests := []struct {
		name string
		args args
		want map[string]int
	}{
		{
			name: "2 new nodes",
			args: args{
				newMasterNodes: redis.Nodes{redis1, redis2, redis3},
				oldMasterNodes: redis.Nodes{redis1},
				allMasterNodes: redis.Nodes{redis1, redis2, redis3},
				nbSlots:        9,
			},
			want: map[string]int{
				redis2.ID: 3,
				redis3.ID: 3,
			},
		},
		{
			name: "1 new node",
			args: args{
				newMasterNodes: redis.Nodes{redis1, redis2},
				oldMasterNodes: redis.Nodes{redis1},
				allMasterNodes: redis.Nodes{redis1, redis2},
				nbSlots:        9,
			},
			want: map[string]int{
				redis2.ID: 4,
			},
		},
		{
			name: "2 new nodes, one removed",
			args: args{
				newMasterNodes: redis.Nodes{redis4, redis2, redis3},
				oldMasterNodes: redis.Nodes{redis4, redis5},
				allMasterNodes: redis.Nodes{redis4, redis2, redis3, redis5},
				nbSlots:        9,
			},
			want: map[string]int{
				redis2.ID: 3,
				redis3.ID: 3,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildSlotsByNode(tt.args.newMasterNodes, tt.args.oldMasterNodes, tt.args.allMasterNodes, tt.args.nbSlots)
			gotSlotByNodeID := make(map[string]int)
			for id, slots := range got {
				gotSlotByNodeID[id] = len(slots)
			}
			if !reflect.DeepEqual(gotSlotByNodeID, tt.want) {
				t.Errorf("buildSlotsByNode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_feedMigInfo(t *testing.T) {
	redis1 := &redis.Node{ID: "redis1", Slots: []redis.Slot{0, 1, 2, 3, 4, 5, 6, 7, 8}}
	redis2 := &redis.Node{ID: "redis2", Slots: []redis.Slot{}}
	redis3 := &redis.Node{ID: "redis3", Slots: []redis.Slot{}}

	type args struct {
		newMasterNodes redis.Nodes
		oldMasterNodes redis.Nodes
		allMasterNodes redis.Nodes
		nbSlots        int
	}
	tests := []struct {
		name       string
		args       args
		wantMapOut mapSlotByMigInfo
	}{
		{
			name: "basic usecase",
			args: args{
				newMasterNodes: redis.Nodes{redis1, redis2, redis3},
				oldMasterNodes: redis.Nodes{redis1},
				allMasterNodes: redis.Nodes{redis1, redis2, redis3},
				nbSlots:        9,
			},
			wantMapOut: mapSlotByMigInfo{
				migrationInfo{From: redis1, To: redis2}: []redis.Slot{3, 4, 5},
				migrationInfo{From: redis1, To: redis3}: []redis.Slot{6, 7, 8},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotMapOut, _ := feedMigInfo(tt.args.newMasterNodes, tt.args.oldMasterNodes, tt.args.allMasterNodes, tt.args.nbSlots); !reflect.DeepEqual(gotMapOut, tt.wantMapOut) {
				t.Errorf("feedMigInfo() = %v, want %v", gotMapOut, tt.wantMapOut)
			}
		})
	}
}
