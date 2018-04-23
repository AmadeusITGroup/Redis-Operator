package sanitycheck

import (
	"fmt"
	"testing"

	"github.com/amadeusitgroup/redis-operator/pkg/redis"
)

func Test_fixNodesNotMeetFunc(t *testing.T) {
	type args struct {
		admin   redis.AdminInterface
		infos   *redis.ClusterInfos
		meetMap map[string]listIds
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "empty node list",
			args: args{
				meetMap: map[string]listIds{},
				infos:   &redis.ClusterInfos{},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "3nodes all know each others",
			args: args{
				meetMap: map[string]listIds{},
				infos: &redis.ClusterInfos{
					Infos: map[string]*redis.NodeInfos{
						"node1": {
							Node:    &redis.Node{ID: "node1"},
							Friends: redis.Nodes{&redis.Node{ID: "node2"}, &redis.Node{ID: "node3"}},
						},
						"node2": {
							Node:    &redis.Node{ID: "node2"},
							Friends: redis.Nodes{&redis.Node{ID: "node1"}, &redis.Node{ID: "node3"}},
						},
						"node3": {
							Node:    &redis.Node{ID: "node3"},
							Friends: redis.Nodes{&redis.Node{ID: "node1"}, &redis.Node{ID: "node2"}},
						},
					},
				},
			},

			want:    false,
			wantErr: false,
		},
		{
			name: "node1 dont know node3, node2 dont know node1",
			args: args{
				meetMap: map[string]listIds{
					"node1": {"node3"},
					"node2": {"node1"},
				},
				infos: &redis.ClusterInfos{
					Infos: map[string]*redis.NodeInfos{
						"node1": {
							Node:    &redis.Node{ID: "node1"},
							Friends: redis.Nodes{&redis.Node{ID: "node2"}},
						},
						"node2": {
							Node:    &redis.Node{ID: "node2"},
							Friends: redis.Nodes{&redis.Node{ID: "node3"}},
						},
						"node3": {
							Node:    &redis.Node{ID: "node3"},
							Friends: redis.Nodes{&redis.Node{ID: "node1"}, &redis.Node{ID: "node2"}},
						},
					},
				},
			},

			want:    true,
			wantErr: false,
		},
		{
			name: "Error: node4 dont exist in admin",
			args: args{
				meetMap: map[string]listIds{},
				infos: &redis.ClusterInfos{
					Infos: map[string]*redis.NodeInfos{
						"node1": {
							Node:    &redis.Node{ID: "node1"},
							Friends: redis.Nodes{&redis.Node{ID: "node2"}, &redis.Node{ID: "node3"}},
						},
						"node2": {
							Node:    &redis.Node{ID: "node2"},
							Friends: redis.Nodes{&redis.Node{ID: "node1"}, &redis.Node{ID: "node3"}},
						},
						"node3": {
							Node:    &redis.Node{ID: "node3"},
							Friends: redis.Nodes{&redis.Node{ID: "node1"}, &redis.Node{ID: "node2"}},
						},
						"node4": {
							Node:    &redis.Node{ID: "node4"},
							Friends: redis.Nodes{},
						},
					},
				},
			},

			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meetTest := meetNodesTest{meet: tt.args.meetMap}
			got, err := fixNodesNotMeetFunc(tt.args.admin, tt.args.infos, meetTest.meetFuncTest, false)
			if (err != nil) != tt.wantErr {
				t.Errorf("fixNodesNotMeetFunc() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("fixNodesNotMeetFunc() = %v, want %v", got, tt.want)
			}
		})
	}
}

type listIds []string
type meetNodesTest struct {
	t    *testing.T
	meet map[string]listIds
}

func (m *meetNodesTest) meetFuncTest(admin redis.AdminInterface, node1, node2 *redis.Node) error {
	list1, ok := m.meet[node1.ID]
	if !ok {
		return fmt.Errorf("unexpected node1:%s", node1.ID)
	}
	found := false
	for _, n := range list1 {
		if n == node2.ID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("unexpected node2:%s", node2.ID)
	}
	return nil
}
