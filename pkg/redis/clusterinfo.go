package redis

import (
	"fmt"
	"net"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
)

const (
	// ClusterInfosUnset status of the cluster info: no data set
	ClusterInfosUnset = "Unset"
	// ClusterInfosPartial status of the cluster info: data is not complete (some nodes didn't respond)
	ClusterInfosPartial = "Partial"
	// ClusterInfosInconsistent status of the cluster info: nodesinfos is not consistent between nodes
	ClusterInfosInconsistent = "Inconsistent"
	// ClusterInfosConsistent status of the cluster info: nodeinfos is complete and consistent between nodes
	ClusterInfosConsistent = "Consistent"
)

// NodeInfos representation of a node info, i.e. data returned by the CLUSTER NODE redis command
// Node is the information of the targetted node
// Friends are the view of the other nodes from the targetted node
type NodeInfos struct {
	Node    *Node
	Friends Nodes
}

// ClusterInfos represents the node infos for all nodes of the cluster
type ClusterInfos struct {
	Infos  map[string]*NodeInfos
	Status string
}

// NewNodeInfos returns an instance of NodeInfo
func NewNodeInfos() *NodeInfos {
	return &NodeInfos{
		Node:    NewDefaultNode(),
		Friends: Nodes{},
	}
}

// NewClusterInfos returns an instance of ClusterInfos
func NewClusterInfos() *ClusterInfos {
	return &ClusterInfos{
		Infos:  make(map[string]*NodeInfos),
		Status: ClusterInfosUnset,
	}
}

// DecodeNodeStartTime decode from the cmd output the Redis instance info. Second argument is the node on which we are connected to request info
func DecodeNodeStartTime(input *string) (time.Time, error) {
	lines := strings.Split(*input, "\n")
	for _, line := range lines {
		values := strings.Split(line, ":")
		if values[0] == "uptime_in_seconds" {
			uptimeInSeconds, err := strconv.Atoi(strings.TrimSpace(values[1]))
			if err != nil {
				glog.Errorf("Error while decoding redis instance uptime in seconds. String : %s Error: %v", values[1], err)
				return time.Now(), err
			}
			return time.Now().Add(-time.Duration(uptimeInSeconds) * time.Second), nil
		}
	}
	glog.Errorf("Error while decoding redis instance uptime in seconds. No data found")
	return time.Now(), fmt.Errorf("Error while decoding redis instance uptime in seconds. No data found")
}

// DecodeNodeInfos decode from the cmd output the Redis nodes info. Second argument is the node on which we are connected to request info
func DecodeNodeInfos(input *string, addr string) *NodeInfos {
	infos := NewNodeInfos()
	lines := strings.Split(*input, "\n")
	for _, line := range lines {
		values := strings.Split(line, " ")
		if len(values) < 8 {
			// last line is always empty
			glog.V(7).Infof("Not enough values in line split, ignoring line: '%s'", line)
			continue
		} else {
			node := NewDefaultNode()

			node.ID = values[0]
			//remove trailing port for cluster internal protocol
			ipPort := strings.Split(values[1], "@")
			if ip, port, err := net.SplitHostPort(ipPort[0]); err == nil {
				node.IP = ip
				node.Port = port
				if ip == "" {
					// ip of the node we are connecting to is sometime empty
					node.IP, _, _ = net.SplitHostPort(addr)
				}
			} else {
				glog.Errorf("Error while decoding node info for node '%s', cannot split ip:port ('%s'): %v", node.ID, values[1], err)
			}
			node.SetRole(values[2])
			node.SetFailureStatus(values[2])
			node.SetReferentMaster(values[3])
			if i, err := strconv.ParseInt(values[4], 10, 64); err == nil {
				node.PingSent = i
			}
			if i, err := strconv.ParseInt(values[5], 10, 64); err == nil {
				node.PongRecv = i
			}
			if i, err := strconv.ParseInt(values[6], 10, 64); err == nil {
				node.ConfigEpoch = i
			}
			node.SetLinkStatus(values[7])

			for _, slot := range values[8:] {
				if s, importing, migrating, err := DecodeSlotRange(slot); err == nil {
					node.Slots = append(node.Slots, s...)
					if importing != nil {
						node.ImportingSlots[importing.SlotID] = importing.FromNodeID
					}
					if migrating != nil {
						node.MigratingSlots[migrating.SlotID] = migrating.ToNodeID
					}
				}
			}

			if strings.HasPrefix(values[2], "myself") {
				infos.Node = node
				glog.V(7).Infof("Getting node info for node: '%s'", node)
			} else {
				infos.Friends = append(infos.Friends, node)
				glog.V(7).Infof("Adding node to slice: '%s'", node)
			}
		}
	}

	return infos
}

// ComputeStatus check the ClusterInfos status based on the current data
// the status ClusterInfosPartial is set while building the clusterinfos
// if already set, do nothing
// returns true if contistent or if another error
func (c *ClusterInfos) ComputeStatus() bool {
	if c.Status != ClusterInfosUnset {
		return false
	}

	consistencyStatus := false

	consolidatedView := c.GetNodes().SortByFunc(LessByID)
	consolidatedSignature := getConfigSignature(consolidatedView)
	glog.V(7).Infof("Consolidated view:\n%s", consolidatedSignature)
	for addr, nodeinfos := range c.Infos {
		nodesView := append(nodeinfos.Friends, nodeinfos.Node).SortByFunc(LessByID)
		nodeSignature := getConfigSignature(nodesView)
		glog.V(7).Infof("Node view from %s (ID: %s):\n%s", addr, nodeinfos.Node.ID, nodeSignature)
		if !reflect.DeepEqual(consolidatedSignature, nodeSignature) {
			glog.V(2).Info("Temporary inconsistency between nodes is possible. If the following inconsistency message persists for more than 20 mins, any cluster operation (scale, rolling update) should be avoided before the message is gone")
			glog.V(2).Infof("Inconsistency from %s: \n%s\nVS\n%s", addr, consolidatedSignature, nodeSignature)
			c.Status = ClusterInfosInconsistent
		}
	}
	if c.Status == ClusterInfosUnset {
		c.Status = ClusterInfosConsistent
		consistencyStatus = true
	}
	return consistencyStatus
}

// GetNodes returns a nodeSlice view of the cluster
// the slice if formed from how each node see itself
// you should check the Status before doing it, to wait for a consistent view
func (c *ClusterInfos) GetNodes() Nodes {
	nodes := Nodes{}
	for _, nodeinfos := range c.Infos {
		nodes = append(nodes, nodeinfos.Node)
	}
	return nodes
}

// ConfigSignature Represents the slots of each node
type ConfigSignature map[string]SlotSlice

// String representation of a ConfigSignaure
func (c ConfigSignature) String() string {
	s := "map["
	sc := make([]string, 0, len(c))
	for i := range c {
		sc = append(sc, i)
	}
	sort.Strings(sc)
	for _, i := range sc {
		s += fmt.Sprintf("%s:%s\n", i, c[i])
	}
	s += "]"
	return s
}

// getConfigSignature returns a way to identify a cluster view, to check consistency
func getConfigSignature(nodes Nodes) ConfigSignature {
	signature := ConfigSignature{}
	for _, node := range nodes {
		if node.Role == redisMasterRole {
			signature[node.ID] = SlotSlice(node.Slots)
		}
	}
	return signature
}

// OwnerWithStatus represents a node owner and the way it sees the slot
type OwnerWithStatus struct {
	Addr   string
	Status string
}

// OwneshipView map representing who owns a slot and who sees it
type OwneshipView map[OwnerWithStatus][]string

// ClusterInconsistencies structure representing inconsistencies in the cluster
type ClusterInconsistencies map[Slot]OwneshipView

// String
func (ci ClusterInconsistencies) String() string {
	output := ""

	for slot, ownership := range ci {
		output += fmt.Sprintf("%d: %s\n", slot, ownership)
	}

	return output
}

// GetInconsistencies returns a view of the inconsistent configuration per slot
func (c *ClusterInfos) GetInconsistencies() *ClusterInconsistencies {
	ci := ClusterInconsistencies{}
	for addr, nodeinfo := range c.Infos {
		allSlots := []Slot{}
		for _, node := range append(nodeinfo.Friends, nodeinfo.Node) {
			// owned slots
			allSlots = append(allSlots, node.Slots...)
			for _, slot := range node.Slots {
				if _, ok := ci[slot]; !ok {
					ci[slot] = OwneshipView{}
				}
				viewers := ci[slot][OwnerWithStatus{Addr: node.IPPort(), Status: "owned"}]
				ci[slot][OwnerWithStatus{Addr: node.IPPort(), Status: "owned"}] = append(viewers, addr)
			}
			// migrating slots
			for slot := range node.MigratingSlots {
				if _, ok := ci[slot]; !ok {
					ci[slot] = OwneshipView{}
				}
				viewers := ci[slot][OwnerWithStatus{Addr: node.IPPort(), Status: "migrating"}]
				ci[slot][OwnerWithStatus{Addr: node.IPPort(), Status: "migrating"}] = append(viewers, addr)
			}
			// importing slots
			for slot := range node.ImportingSlots {
				if _, ok := ci[slot]; !ok {
					ci[slot] = OwneshipView{}
				}
				viewers := ci[slot][OwnerWithStatus{Addr: node.IPPort(), Status: "importing"}]
				ci[slot][OwnerWithStatus{Addr: node.IPPort(), Status: "importing"}] = append(viewers, addr)
			}
		}
		// slots that are not owned according to this node
		missing := RemoveSlots(BuildSlotSlice(0, 16383), allSlots)
		for _, slot := range missing {
			if _, ok := ci[slot]; !ok {
				ci[slot] = OwneshipView{}
			}
			viewers := ci[slot][OwnerWithStatus{Addr: "", Status: ""}]
			ci[slot][OwnerWithStatus{Addr: "", Status: ""}] = append(viewers, addr)
		}
	}
	// now cleaning all consistent data
	for slot, ownership := range ci {
		if len(ownership) <= 1 {
			delete(ci, slot)
		}
	}

	return &ci
}
