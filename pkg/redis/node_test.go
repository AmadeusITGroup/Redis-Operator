package redis

import (
	"reflect"
	"sort"
	"testing"

	"github.com/amadeusitgroup/redis-operator/pkg/api/redis/v1"
	kapiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	pod1  = &kapiv1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "Pod1", Namespace: "ns"}}
	pod2  = &kapiv1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "Pod2", Namespace: "ns"}}
	pod3  = &kapiv1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "Pod3", Namespace: "ns"}}
	pod4  = &kapiv1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "Pod4", Namespace: "ns"}}
	node1 = NewNode("abcd", "1.2.3.1", pod1)
	node2 = NewNode("edfg", "1.2.3.2", pod2)
	node3 = NewNode("igkl", "1.2.3.3", pod3)
	node4 = NewNode("mnop", "1.2.3.4", pod4)
)

func TestNode_ToAPINode(t *testing.T) {
	type fields struct {
		ID             string
		IP             string
		MasterReferent string
		Slots          []Slot
		Pod            *kapiv1.Pod
	}
	tests := []struct {
		name   string
		fields fields
		want   v1.RedisClusterNode
	}{
		{
			name: "default test",
			fields: fields{
				ID: "id1",
				IP: "1.2.3.4",
				Pod: &kapiv1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "name1", Namespace: "NS1"},
				},
				Slots: []Slot{},
			},
			want: v1.RedisClusterNode{
				ID:      "id1",
				IP:      "1.2.3.4",
				PodName: "name1",
				Role:    v1.RedisClusterNodeRoleNone,
				Slots:   []string{},
			},
		},
		{
			name: "convert a master",
			fields: fields{
				ID: "id1",
				IP: "1.2.3.4",
				Pod: &kapiv1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "name1", Namespace: "NS1"},
				},
				Slots: []Slot{Slot(1), Slot(2)},
			},
			want: v1.RedisClusterNode{
				ID:      "id1",
				IP:      "1.2.3.4",
				PodName: "name1",
				Role:    v1.RedisClusterNodeRoleMaster,
				Slots:   []string{},
			},
		},
		{
			name: "convert a slave",
			fields: fields{
				ID: "id1",
				IP: "1.2.3.4",
				Pod: &kapiv1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "name1", Namespace: "NS1"},
				},
				MasterReferent: "idMaster",
				Slots:          []Slot{},
			},
			want: v1.RedisClusterNode{
				ID:      "id1",
				IP:      "1.2.3.4",
				PodName: "name1",
				Role:    v1.RedisClusterNodeRoleSlave,
				Slots:   []string{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &Node{
				ID:             tt.fields.ID,
				IP:             tt.fields.IP,
				Pod:            tt.fields.Pod,
				MasterReferent: tt.fields.MasterReferent,
				Slots:          tt.fields.Slots,
			}
			if got := n.ToAPINode(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Node.ToAPINode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodes_SortNodes(t *testing.T) {
	sortedNodes := Nodes{node1, node2, node3, node4}
	sort.Sort(sortedNodes)
	unsertedNodes := Nodes{node4, node3, node2, node1}

	tests := []struct {
		name string
		ns   Nodes
		want Nodes
	}{
		{
			name: "empty nodes",
			ns:   Nodes{},
			want: Nodes{},
		},
		{
			name: "already sorted nodes",
			ns:   sortedNodes,
			want: sortedNodes,
		},
		{
			name: "unsorted nodes",
			ns:   unsertedNodes,
			want: sortedNodes,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ns.SortNodes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Nodes.SortNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeSetRoleMasterValid(t *testing.T) {
	node := &Node{}

	flags := "master"
	err := node.SetRole(flags)
	if err != nil {
		t.Error("Failed to set Master as role [err]:", err)
	}
	if node.Role != redisMasterRole {
		t.Error("Role should be Master")
	}
}

func TestNodeSetRoleSlaveValid(t *testing.T) {
	node := &Node{}

	flags := "slave"
	err := node.SetRole(flags)
	if err != nil {
		t.Error("Failed to set Slave as role [err]:", err)
	}
	if node.Role != redisSlaveRole {
		t.Error("Role should be Slave")
	}
}

func TestNodeSetRoleNotValid(t *testing.T) {
	node := &Node{}

	flags := "king"
	err := node.SetRole(flags)
	if err == nil {
		t.Error("SetRole should return an error ")
	}

	if node.Role != "" {
		t.Error("Role should be empty current:", node.Role)
	}
}

func TestNodeSetRoleMultFlags(t *testing.T) {
	node := &Node{}

	flags := "myself,slave"
	err := node.SetRole(flags)
	if err != nil {
		t.Error("Failed to set Slave as role [err]:", err)
	}
	if node.Role != redisSlaveRole {
		t.Error("Role should be Slave")
	}
}

func TestNodeSetLinkStatusConnected(t *testing.T) {
	node := &Node{}

	status := "connected"
	err := node.SetLinkStatus(status)
	if err != nil {
		t.Error("Failed to set link status [err]:", err)
	}
	if node.LinkState != RedisLinkStateConnected {
		t.Error("State should be connected")
	}
}

func TestNodeSetLinkStatusDisconnected(t *testing.T) {
	node := &Node{}

	status := "disconnected"
	err := node.SetLinkStatus(status)
	if err != nil {
		t.Error("Failed to set link status [err]:", err)
	}
	if node.LinkState != RedisLinkStateDisconnected {
		t.Error("State should be disconnected")
	}
}

func TestNodeSetLinkStatusKO(t *testing.T) {
	node := &Node{}

	status := "blabla"
	err := node.SetLinkStatus(status)
	if err == nil {
		t.Error("SetLinkStatus should return an error ")
	}

	if node.LinkState != "" {
		t.Error("State should be empty current:", node.LinkState)
	}
}

func TestNodeSetFailureStateFail(t *testing.T) {
	node := &Node{}

	flags := "master,myself,fail"
	node.SetFailureStatus(flags)

	if !node.HasStatus(NodeStatusFail) {
		t.Error("Failure Status should be NodeStatusFail current:", node.FailStatus)
	}
}

func TestNodeSetFailureStatePFail(t *testing.T) {
	node := &Node{}

	flags := "master,myself,fail?"
	node.SetFailureStatus(flags)

	if !node.HasStatus(NodeStatusPFail) {
		t.Error("Failure Status should be NodeStatusFail current:", node.FailStatus)
	}
}

func TestNodeSetFailureStateOK(t *testing.T) {
	node := &Node{}

	flags := "master,myself"
	node.SetFailureStatus(flags)

	if len(node.FailStatus) > 0 {
		t.Error("Failure Status should be empty current:", node.FailStatus)
	}
}

func TestNodeSliceTestSearchInSlde(t *testing.T) {
	node := &Node{}

	flags := "master,myself"
	node.SetFailureStatus(flags)

	if len(node.FailStatus) > 0 {
		t.Error("Failure Status should be empty current:", node.FailStatus)
	}
}

func TestNodeSetReferentMaster(t *testing.T) {
	node := &Node{}

	ref := "899809809808343434342323"
	node.SetReferentMaster(ref)
	if node.MasterReferent != ref {
		t.Error("Node MasterReferent is not correct [current]:", node.MasterReferent)
	}
}

func TestNodeSetReferentMasterNone(t *testing.T) {
	node := &Node{}

	ref := "-"
	node.SetReferentMaster(ref)
	if node.MasterReferent != "" {
		t.Error("Node MasterReferent should be empty  [current]:", node.MasterReferent)
	}
}
func TestNodeWhereP(t *testing.T) {
	var slice Nodes
	nodeMaster := &Node{ID: "A", Role: redisMasterRole, Slots: []Slot{0, 1, 4, 10}}
	slice = append(slice, nodeMaster)
	nodeSlave := &Node{ID: "B", Role: redisSlaveRole, Slots: []Slot{}}
	slice = append(slice, nodeSlave)
	nodeUnset := &Node{ID: "C", Role: redisMasterRole, Slots: []Slot{}}
	slice = append(slice, nodeUnset)

	masterSlice, err := slice.GetNodesByFunc(IsMasterWithSlot)
	if err != nil {
		t.Error("slice.GetNodesByFunc(IsMasterWithSlot) sould not return an error, current err:", err)
	}
	if len(masterSlice) != 1 {
		t.Error("masterSlice should have a size of 1, current:", len(masterSlice))
	}
	if masterSlice[0].ID != "A" {
		t.Error("masterSlice[0].ID should be A current:", masterSlice[0].ID)
	}

	unsetSlice, err := slice.GetNodesByFunc(IsMasterWithNoSlot)
	if err != nil {
		t.Error("slice.GetNodesByFunc(IsMasterWithSlot) sould not return an error, current err:", err)
	}
	if len(unsetSlice) != 1 {
		t.Error("unsetSlice should have a size of 1, current:", len(unsetSlice))
	}
	if unsetSlice[0].ID != "C" {
		t.Error("unsetSlice[0].ID should should be C current:", unsetSlice[0].ID)
	}

	slaveSlice, err := slice.GetNodesByFunc(IsSlave)
	if err != nil {
		t.Error("slice.GetNodesByFunc(IsMasterWithSlot) sould not return an error, current err:", err)
	}
	if len(slaveSlice) != 1 {
		t.Error("slaveSlice should have a size of 1, current:", len(slaveSlice))
	}
	if slaveSlice[0].ID != "B" {
		t.Error("slaveSlice[0].ID should should be B current:", slaveSlice[0].ID)
	}
}

func TestSearchNodeByID(t *testing.T) {
	var slice Nodes
	nodeMaster := &Node{ID: "A", Role: redisMasterRole, Slots: []Slot{0, 1, 4, 10}}
	slice = append(slice, nodeMaster)
	nodeSlave := &Node{ID: "B", Role: redisSlaveRole, Slots: []Slot{}}
	slice = append(slice, nodeSlave)
	nodeUnset := &Node{ID: "C", Role: redisMasterRole, Slots: []Slot{}}
	slice = append(slice, nodeUnset)

	// empty list
	_, err := Nodes{}.GetNodeByID("B")
	if err == nil {
		t.Errorf("With an empty list, GetNodeByID should return an error")
	}

	// empty list
	_, err = slice.GetNodeByID("D")
	if err == nil {
		t.Errorf("The Node D is not present in the list, GetNodeByID should return an error")
	}

	// not empty
	node, err := slice.GetNodeByID("B")
	if err != nil {
		t.Errorf("Unexpected error returned by GetNodeByID, current error:%v", err)
	}
	if node != nodeSlave {
		t.Errorf("Expected to find node %v, got %v", nodeSlave, node)
	}
}
