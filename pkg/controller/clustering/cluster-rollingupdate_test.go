package clustering

import (
	"reflect"
	"testing"

	"github.com/amadeusitgroup/redis-operator/pkg/redis"
)

func TestReplaceMasters(t *testing.T) {
	node1 := &redis.Node{ID: "node1"}
	node2 := &redis.Node{ID: "node2"}
	node3 := &redis.Node{ID: "node3"}

	newNode1 := &redis.Node{ID: "newNode1"}

	type args struct {
		currentOldMaster  redis.Nodes
		currentNewMasters redis.Nodes
		newNoneNodes      redis.Nodes
		nbMaster          int32
		nbMasterToReplace int32
	}
	tests := []struct {
		name    string
		args    args
		want    redis.Nodes
		want2   redis.Nodes
		wantErr bool
	}{
		{
			name: "empty slices",
			args: args{
				currentOldMaster:  redis.Nodes{},
				currentNewMasters: redis.Nodes{},
				newNoneNodes:      redis.Nodes{},
				nbMaster:          0,
				nbMasterToReplace: 0,
			},
			want:    redis.Nodes{},
			want2:   redis.Nodes{},
			wantErr: false,
		},
		{
			name: "no master to replace",
			args: args{
				currentOldMaster:  redis.Nodes{node1, node2, node3},
				currentNewMasters: redis.Nodes{},
				newNoneNodes:      redis.Nodes{},
				nbMaster:          3,
				nbMasterToReplace: 0,
			},
			want:    redis.Nodes{node1, node2, node3},
			want2:   redis.Nodes{},
			wantErr: false,
		},
		{
			name: "one master to replace",
			args: args{
				currentOldMaster:  redis.Nodes{node1, node2, node3},
				currentNewMasters: redis.Nodes{},
				newNoneNodes:      redis.Nodes{newNode1},
				nbMaster:          3,
				nbMasterToReplace: 1,
			},
			want:    redis.Nodes{node1, node2, newNode1},
			want2:   redis.Nodes{newNode1},
			wantErr: false,
		},
		{
			name: "one master to replace, current Master as already one master migrated",
			args: args{
				currentOldMaster:  redis.Nodes{node1, node2},
				currentNewMasters: redis.Nodes{node3},
				newNoneNodes:      redis.Nodes{newNode1},
				nbMaster:          3,
				nbMasterToReplace: 1,
			},
			want:    redis.Nodes{node1, node3, newNode1},
			want2:   redis.Nodes{newNode1},
			wantErr: false,
		},
		{
			name: "not enough new nodes",
			args: args{
				currentOldMaster:  redis.Nodes{node1, node2, node3},
				currentNewMasters: redis.Nodes{},
				newNoneNodes:      redis.Nodes{newNode1},
				nbMaster:          3,
				nbMasterToReplace: 2,
			},
			want:    redis.Nodes{node1, node2, newNode1},
			want2:   redis.Nodes{newNode1},
			wantErr: true,
		},
		{
			name: "not enough masters",
			args: args{
				currentOldMaster:  redis.Nodes{node1, node2, node3},
				currentNewMasters: redis.Nodes{},
				newNoneNodes:      redis.Nodes{newNode1},
				nbMaster:          5,
				nbMasterToReplace: 1,
			},
			want:    redis.Nodes{node1, node2, node3, newNode1},
			want2:   redis.Nodes{newNode1},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got2, err := SelectMastersToReplace(tt.args.currentOldMaster, tt.args.currentNewMasters, tt.args.newNoneNodes, tt.args.nbMaster, tt.args.nbMasterToReplace)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReplaceMasters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			sortedGot := got.SortNodes()
			sortedGot2 := got2.SortNodes()
			sortedWant := tt.want.SortNodes()
			sortedWant2 := tt.want2.SortNodes()
			if !reflect.DeepEqual(sortedGot, sortedWant) {
				t.Errorf("ReplaceMasters().selectedMasters = %v, want %v", sortedGot, sortedWant)
			}
			if !reflect.DeepEqual(sortedGot2, sortedWant2) {
				t.Errorf("ReplaceMasters().newSelectedMasters = %v, want %v", sortedGot2, sortedWant2)
			}
		})
	}
}
