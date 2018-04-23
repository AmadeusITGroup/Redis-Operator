package clustering

import (
	"reflect"
	"testing"

	"github.com/amadeusitgroup/redis-operator/pkg/redis"
)

func TestPlaceMasters(t *testing.T) {
	redisNode1 := &redis.Node{ID: "1", Role: "master", IP: "1.1.1.1", Port: "1234", Slots: []redis.Slot{}, Pod: newPod("pod1", "vm1")}
	redisNode2 := &redis.Node{ID: "2", Role: "master", IP: "1.1.1.2", Port: "1234", Slots: []redis.Slot{}, Pod: newPod("pod2", "vm1")}
	redisNode3 := &redis.Node{ID: "3", Role: "master", IP: "1.1.1.3", Port: "1234", Slots: []redis.Slot{}, Pod: newPod("pod3", "vm2")}
	redisNode4 := &redis.Node{ID: "4", Role: "master", IP: "1.1.1.4", Port: "1234", Slots: []redis.Slot{}, Pod: newPod("pod4", "vm3")}

	redisNode2Vm2 := &redis.Node{ID: "2", Role: "master", IP: "1.1.1.2", Port: "1234", Slots: []redis.Slot{}, Pod: newPod("pod2", "vm2")}
	redisNode4Vm2 := &redis.Node{ID: "4", Role: "master", IP: "1.1.1.4", Port: "1234", Slots: []redis.Slot{}, Pod: newPod("pod4", "vm2")}

	type args struct {
		cluster            *redis.Cluster
		currentMaster      redis.Nodes
		allPossibleMasters redis.Nodes
		nbMaster           int32
	}
	tests := []struct {
		name           string
		args           args
		want           redis.Nodes
		wantBestEffort bool
		wantError      bool
	}{
		{
			name: "found 3 master on 3 vms, discard pod2 since pod1 and pod2 are on the same vm ",
			args: args{
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
				currentMaster:      redis.Nodes{},
				allPossibleMasters: redis.Nodes{redisNode1, redisNode2, redisNode3, redisNode4},
				nbMaster:           3,
			},
			want:           redis.Nodes{redisNode1, redisNode3, redisNode4},
			wantBestEffort: false,
			wantError:      false,
		},
		{
			name: "Best effort mode, 4masters on 2 vms",
			args: args{
				cluster: &redis.Cluster{
					Name:      "clustertest",
					Namespace: "default",
					Nodes: map[string]*redis.Node{
						"1": redisNode1,
						"2": redisNode2,
						"3": redisNode3,
						"4": redisNode4Vm2,
					},
				},
				currentMaster:      redis.Nodes{},
				allPossibleMasters: redis.Nodes{redisNode1, redisNode2, redisNode3, redisNode4Vm2},
				nbMaster:           4,
			},
			want:           redis.Nodes{redisNode1, redisNode2, redisNode3, redisNode4Vm2},
			wantBestEffort: true,
			wantError:      false,
		},
		{
			name: "error case not enough node",
			args: args{
				cluster: &redis.Cluster{
					Name:      "clustertest",
					Namespace: "default",
					Nodes: map[string]*redis.Node{
						"1": redisNode1,
						"2": redisNode2Vm2,
					},
				},
				currentMaster:      redis.Nodes{},
				allPossibleMasters: redis.Nodes{redisNode1, redisNode2Vm2},
				nbMaster:           3,
			},
			want:           redis.Nodes{redisNode1, redisNode2Vm2},
			wantBestEffort: true,
			wantError:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.args.cluster
			gotNodes, gotBestEffort, gotError := PlaceMasters(c, tt.args.currentMaster, tt.args.allPossibleMasters, tt.args.nbMaster)
			gotNodes = gotNodes.SortNodes()
			tt.want = tt.want.SortNodes()
			if gotBestEffort != tt.wantBestEffort {
				t.Errorf("SmartMastersPlacement() best effort :%v, want:%v", gotBestEffort, tt.wantBestEffort)
			}
			if (gotError != nil && !tt.wantError) || (tt.wantError && gotError == nil) {
				t.Errorf("SmartMastersPlacement() return error:%v, want:%v", gotError, tt.wantError)
			}
			if !reflect.DeepEqual(gotNodes, tt.want) {
				t.Errorf("SmartMastersPlacement() = %v, want %v", gotNodes, tt.want)
			}
		})
	}
}
