package redis

import (
	"reflect"
	"sort"
	"testing"
)

func TestCluster_GetNodeByID(t *testing.T) {

	type fields struct {
		Name      string
		Namespace string
		Nodes     map[string]*Node
	}
	type args struct {
		id string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *Node
		wantErr bool
	}{
		{
			name: "Node found",
			fields: fields{"Cluster", "ns", map[string]*Node{
				node1.ID: node1,
				node2.ID: node2,
				node3.ID: node3,
				node4.ID: node4}},
			args:    args{id: node4.ID},
			want:    node4,
			wantErr: false,
		},
		{
			name: "Node not found",
			fields: fields{"Cluster", "ns", map[string]*Node{
				node1.ID: node1,
				node2.ID: node2,
				node3.ID: node3,
				node4.ID: node4}},
			args:    args{id: "sdsd"},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cluster{
				Name:      tt.fields.Name,
				Namespace: tt.fields.Namespace,
				Nodes:     tt.fields.Nodes,
			}
			got, err := c.GetNodeByID(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("Cluster.GetNodeByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Cluster.GetNodeByID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCluster_GetNodeByIP(t *testing.T) {
	type fields struct {
		Name      string
		Namespace string
		Nodes     map[string]*Node
	}
	type args struct {
		ip string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *Node
		wantErr bool
	}{
		{
			name: "Node found",
			fields: fields{"Cluster", "ns", map[string]*Node{
				node1.ID: node1,
				node2.ID: node2,
				node3.ID: node3,
				node4.ID: node4}},
			args:    args{ip: node4.IP},
			want:    node4,
			wantErr: false,
		},
		{
			name: "Node not found",
			fields: fields{"Cluster", "ns", map[string]*Node{
				node1.ID: node1,
				node2.ID: node2,
				node3.ID: node3,
				node4.ID: node4}},
			args:    args{ip: "1.2.3.6"},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cluster{
				Name:      tt.fields.Name,
				Namespace: tt.fields.Namespace,
				Nodes:     tt.fields.Nodes,
			}
			got, err := c.GetNodeByIP(tt.args.ip)
			if (err != nil) != tt.wantErr {
				t.Errorf("Cluster.GetNodeByIP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Cluster.GetNodeByIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCluster_GetNodesByFunc(t *testing.T) {
	type fields struct {
		Name      string
		Namespace string
		Nodes     map[string]*Node
	}
	type args struct {
		f FindNodeFunc
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    Nodes
		wantErr bool
	}{
		{
			name: "Node found",
			fields: fields{"Cluster", "ns", map[string]*Node{
				node1.ID: node1,
				node2.ID: node2,
				node3.ID: node3,
				node4.ID: node4}},
			args: args{f: func(node *Node) bool {
				return node.Pod.Namespace == "ns"
			}},
			want:    Nodes{node1, node2, node3, node4},
			wantErr: false,
		},
		{
			name: "Node not found",
			fields: fields{"Cluster", "ns", map[string]*Node{
				node1.ID: node1,
				node2.ID: node2,
				node3.ID: node3,
				node4.ID: node4}},
			args: args{f: func(node *Node) bool {
				return node.Pod.Namespace == "default"
			}},
			want:    Nodes{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cluster{
				Name:      tt.fields.Name,
				Namespace: tt.fields.Namespace,
				Nodes:     tt.fields.Nodes,
			}
			got, err := c.GetNodesByFunc(tt.args.f)
			if (err != nil) != tt.wantErr {
				t.Errorf("Cluster.GetNodesByFunc() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			sort.Sort(tt.want)
			sort.Sort(got)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Cluster.GetNodesByFunc() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCluster_GetNodeByPodName(t *testing.T) {
	type fields struct {
		Name      string
		Namespace string
		Nodes     map[string]*Node
	}
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *Node
		wantErr bool
	}{
		{
			name: "Node found",
			fields: fields{"Cluster", "ns", map[string]*Node{
				node1.ID: node1,
				node2.ID: node2,
				node3.ID: node3,
				node4.ID: node4}},
			args:    args{name: node4.Pod.Name},
			want:    node4,
			wantErr: false,
		},
		{
			name: "Node not found",
			fields: fields{"Cluster", "ns", map[string]*Node{
				node1.ID: node1,
				node2.ID: node2,
				node3.ID: node3,
				node4.ID: node4}},
			args:    args{name: "blabla"},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cluster{
				Name:      tt.fields.Name,
				Namespace: tt.fields.Namespace,
				Nodes:     tt.fields.Nodes,
			}
			got, err := c.GetNodeByPodName(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("Cluster.GetNodeByPodName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Cluster.GetNodeByPodName() = %v, want %v", got, tt.want)
			}
		})
	}
}
