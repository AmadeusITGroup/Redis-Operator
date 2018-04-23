package fake

// ClusterSlotsNode Representation of a node as returned by the CLUSTER SLOT command
type ClusterSlotsNode struct {
	IP   string
	Port int
}

// ClusterSlotsSlot Representation of a slot as returned by the CLUSTER SLOT command
type ClusterSlotsSlot struct {
	Min   int
	Max   int
	Nodes []ClusterSlotsNode
}
