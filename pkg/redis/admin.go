package redis

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/golang/glog"
)

const (
	// defaultHashMaxSlots higher value of slot
	// as slots start at 0, total number of slots is defaultHashMaxSlots+1
	defaultHashMaxSlots = 16383

	// ResetHard HARD mode for RESET command
	ResetHard = "HARD"
	// ResetSoft SOFT mode for RESET command
	ResetSoft = "SOFT"
)

// AdminInterface redis cluster admin interface
type AdminInterface interface {
	// Connections returns the connection map of all clients
	Connections() AdminConnectionsInterface
	// Close the admin connections
	Close()
	// InitRedisCluster used to configure the first node of a cluster
	InitRedisCluster(addr string) error
	// GetClusterInfos get node infos for all nodes
	GetClusterInfos() (*ClusterInfos, error)
	// GetClusterInfosSelected return the Nodes infos for all nodes selected in the cluster
	GetClusterInfosSelected(addrs []string) (*ClusterInfos, error)
	// AttachNodeToCluster command use to connect a Node to the cluster
	// the connection will be done on a random node part of the connection pool
	AttachNodeToCluster(addr string) error
	// AttachSlaveToMaster attach a slave to a master node
	AttachSlaveToMaster(slave *Node, master *Node) error
	// DetachSlave dettach a slave to its master
	DetachSlave(slave *Node) error
	// StartFailover execute the failover of the Redis Master corresponding to the addr
	StartFailover(addr string) error
	// ForgetNode execute the Redis command to force the cluster to forgot the the Node
	ForgetNode(id string) error
	// ForgetNodeByAddr execute the Redis command to force the cluster to forgot the the Node
	ForgetNodeByAddr(id string) error
	// SetSlots exect the redis command to set slots in a pipeline, provide
	// and empty nodeID if the set slots commands doesn't take a nodeID in parameter
	SetSlots(addr string, action string, slots []Slot, nodeID string) error
	// AddSlots exect the redis command to add slots in a pipeline
	AddSlots(addr string, slots []Slot) error
	// DelSlots exec the redis command to del slots in a pipeline
	DelSlots(addr string, slots []Slot) error
	// GetKeysInSlot exec the redis command to get the keys in the given slot on the node we are connected to
	GetKeysInSlot(addr string, slot Slot, batch int, limit bool) ([]string, error)
	// CountKeysInSlot exec the redis command to count the keys given slot on the node
	CountKeysInSlot(addr string, slot Slot) (int64, error)
	// MigrateKeys from addr to destination node. returns number of slot migrated. If replace is true, replace key on busy error
	MigrateKeys(addr string, dest *Node, slots []Slot, batch, timeout int, replace bool) (int, error)
	// FlushAndReset reset the cluster configuration of the node, the node is flushed in the same pipe to ensure reset works
	FlushAndReset(addr string, mode string) error
	// FlushAll flush all keys in cluster
	FlushAll()
	// GetHashMaxSlot get the max slot value
	GetHashMaxSlot() Slot
	//RebuildConnectionMap rebuild the connection map according to the given addresses
	RebuildConnectionMap(addrs []string, options *AdminOptions)
}

// AdminOptions optional options for redis admin
type AdminOptions struct {
	ConnectionTimeout  time.Duration
	ClientName         string
	RenameCommandsFile string
}

// Admin wraps redis cluster admin logic
type Admin struct {
	hashMaxSlots Slot
	cnx          AdminConnectionsInterface
}

// NewAdmin returns new AdminInterface instance
// at the same time it connects to all Redis Nodes thanks to the addrs list
func NewAdmin(addrs []string, options *AdminOptions) AdminInterface {
	a := &Admin{
		hashMaxSlots: defaultHashMaxSlots,
	}

	// perform initial connections
	a.cnx = NewAdminConnections(addrs, options)

	return a
}

// Connections returns the connection map of all clients
func (a *Admin) Connections() AdminConnectionsInterface {
	return a.cnx
}

// Close used to close all possible resources instanciate by the Admin
func (a *Admin) Close() {
	a.Connections().Reset()
}

// GetHashMaxSlot get the max slot value
func (a *Admin) GetHashMaxSlot() Slot {
	return a.hashMaxSlots
}

// AttachNodeToCluster command use to connect a Node to the cluster
func (a *Admin) AttachNodeToCluster(addr string) error {
	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}

	all := a.Connections().GetAll()
	if len(all) == 0 {
		return fmt.Errorf("no connection for other redis-node found")
	}
	for cAddr, c := range a.Connections().GetAll() {
		if cAddr == addr {
			continue
		}
		resp := c.Cmd("CLUSTER", "MEET", ip, port)
		if err = a.Connections().ValidateResp(resp, addr, "Cannot attach node to cluster"); err != nil {
			return err
		}
	}

	a.Connections().Add(addr)

	glog.Infof("Node %s attached properly", addr)
	return nil
}

// InitRedisCluster used to init a single node redis cluster
func (a *Admin) InitRedisCluster(addr string) error {
	return a.AddSlots(addr, BuildSlotSlice(0, a.GetHashMaxSlot()))
}

// GetClusterInfos return the Nodes infos for all nodes
func (a *Admin) GetClusterInfos() (*ClusterInfos, error) {
	infos := NewClusterInfos()
	clusterErr := NewClusterInfosError()

	for addr, c := range a.Connections().GetAll() {
		nodeinfos, err := a.getInfos(c, addr)
		if err != nil {
			infos.Status = ClusterInfosPartial
			clusterErr.partial = true
			clusterErr.errs[addr] = err
			continue
		}
		if nodeinfos.Node != nil && nodeinfos.Node.IPPort() == addr {
			infos.Infos[addr] = nodeinfos
		} else {
			glog.Warningf("Bad node info retreived from %s", addr)
		}
	}

	if len(clusterErr.errs) == 0 {
		clusterErr.inconsistent = !infos.ComputeStatus()
	}
	if infos.Status == ClusterInfosConsistent {
		return infos, nil
	}
	return infos, clusterErr
}

//GetClusterInfosSelected return the Nodes infos for all nodes selected in the cluster
func (a *Admin) GetClusterInfosSelected(addrs []string) (*ClusterInfos, error) {
	infos := NewClusterInfos()
	clusterErr := NewClusterInfosError()

	for addr, c := range a.Connections().GetSelected(addrs) {
		nodeinfos, err := a.getInfos(c, addr)
		if err != nil {
			infos.Status = ClusterInfosPartial
			clusterErr.partial = true
			clusterErr.errs[addr] = err
			continue
		}
		if nodeinfos.Node != nil && nodeinfos.Node.IPPort() == addr {
			infos.Infos[addr] = nodeinfos
		} else {
			glog.Warningf("Bad node info retreived from %s", addr)
		}
	}

	if len(clusterErr.errs) == 0 {
		clusterErr.inconsistent = !infos.ComputeStatus()
	}
	if infos.Status == ClusterInfosConsistent {
		return infos, nil
	}
	return infos, clusterErr
}

// StartFailover used to force the failover of a specific redis master node
func (a *Admin) StartFailover(addr string) error {
	c, err := a.Connections().Get(addr)
	if err != nil {
		return err
	}
	var me *NodeInfos
	me, err = a.getInfos(c, addr)
	if err != nil {
		return err
	}

	if me.Node.Role != redisMasterRole {
		// if not a Master dont failover
		return nil
	}

	slaves, err := selectMySlaves(me.Node, me.Friends)
	if err != nil {
		return fmt.Errorf("Unable to found associated slaves, err:%s", err)
	}

	if len(slaves) == 0 {
		return fmt.Errorf("Master id:%s dont have associated slave", me.Node.ID)
	}

	if glog.V(3) {
		for _, slave := range slaves {
			glog.Info("- Slave: ", slave.ID)
		}
	}

	failoverTriggered := false
	for _, aSlave := range slaves {
		var slaveClient ClientInterface
		if slaveClient, err = a.Connections().Get(aSlave.IPPort()); err != nil {
			continue
		}

		resp := slaveClient.Cmd("CLUSTER", "FAILOVER")
		if err = a.Connections().ValidateResp(resp, aSlave.IPPort(), "Unable to execute Failover"); err != nil {
			continue
		}
		failoverTriggered = true
		break
	}

	if !failoverTriggered {
		return fmt.Errorf("Unable to trigger failover for node '%s'", me.Node.IPPort())
	}

	for {
		me, err = a.getInfos(c, addr)
		if err != nil {
			return err
		}

		if me.Node.TotalSlots() == 0 {
			glog.Info("failover completed")
			break
		}

		glog.Info("waiting failover to be complete...")
		time.Sleep(time.Second) // TODO: implement back-off like logic
		// we should wait that all slots have been moved to the new master
		// this is the only way to know that we can stop this master with no impact on the cluster
	}

	return nil
}

// ForgetNode used to force other redis cluster node to forget a specific node
func (a *Admin) ForgetNode(id string) error {
	infos, _ := a.GetClusterInfos()
	for nodeAddr, nodeinfos := range infos.Infos {
		if nodeinfos.Node.ID == id {
			continue
		}
		c, err := a.Connections().Get(nodeAddr)
		if err != nil {
			glog.Errorf("Cannot force a forget on node %s, for node %s: %v", nodeAddr, id, err)
			continue
		}

		if IsSlave(nodeinfos.Node) && nodeinfos.Node.MasterReferent == id {
			a.DetachSlave(nodeinfos.Node)
			glog.V(2).Infof("detach slave id: %s of master: %d", nodeinfos.Node.ID, id)
		}

		resp := c.Cmd("CLUSTER", "FORGET", id)
		a.Connections().ValidateResp(resp, nodeAddr, "Unable to execute FORGET command")
	}

	glog.Infof("Forget Node:%s ...done", id)
	return nil
}

// ForgetNodeByAddr used to force other redis cluster node to forget a specific node
func (a *Admin) ForgetNodeByAddr(addr string) error {
	infos, _ := a.GetClusterInfos()
	var me *Node
	myinfo, ok := infos.Infos[addr]
	if !ok {
		// get its id from a random node that still knows it
		for _, nodeinfos := range infos.Infos {
			for _, node := range nodeinfos.Friends {
				if node.IPPort() == addr {
					me = node
					break
				}
			}
			if me != nil {
				break
			}
		}
	} else {
		me = myinfo.Node
	}

	if me == nil {
		return fmt.Errorf("Cannot forget node %s, not found in infos", addr)
	}

	return a.ForgetNode(me.ID)
}

// SetSlots use to set SETSLOT command on several slots
func (a *Admin) SetSlots(addr, action string, slots []Slot, nodeID string) error {
	if len(slots) == 0 {
		return nil
	}
	c, err := a.Connections().Get(addr)
	if err != nil {
		return err
	}
	for _, slot := range slots {
		if nodeID == "" {
			c.PipeAppend("CLUSTER", "SETSLOT", slot, action)
		} else {
			c.PipeAppend("CLUSTER", "SETSLOT", slot, action, nodeID)
		}
	}
	if !a.Connections().ValidatePipeResp(c, addr, "Cannot SETSLOT") {
		return fmt.Errorf("Error occured during CLUSTER SETSLOT %s", action)
	}
	c.PipeClear()

	return nil
}

// AddSlots use to ADDSLOT commands on several slots
func (a *Admin) AddSlots(addr string, slots []Slot) error {
	if len(slots) == 0 {
		return nil
	}
	c, err := a.Connections().Get(addr)
	if err != nil {
		return err
	}

	resp := c.Cmd("CLUSTER", "ADDSLOTS", slots)

	return a.Connections().ValidateResp(resp, addr, "Unable to run CLUSTER ADDSLOTS")
}

// DelSlots exec the redis command to del slots in a pipeline
func (a *Admin) DelSlots(addr string, slots []Slot) error {
	if len(slots) == 0 {
		return nil
	}
	c, err := a.Connections().Get(addr)
	if err != nil {
		return err
	}

	resp := c.Cmd("CLUSTER", "DELSLOTS", slots)

	return a.Connections().ValidateResp(resp, addr, "Unable to run CLUSTER DELSLOTS")
}

// GetKeysInSlot exec the redis command to get the keys in the given slot on the node we are connected to
// Batch is the number of keys fetch per batch, Limit can be use to limit to one batch
func (a *Admin) GetKeysInSlot(addr string, slot Slot, batch int, limit bool) ([]string, error) {
	keyCount := 0
	allKeys := []string{}
	c, err := a.Connections().Get(addr)
	if err != nil {
		return allKeys, err
	}

	for {
		resp := c.Cmd("CLUSTER", "GETKEYSINSLOT", slot, strconv.Itoa(batch))
		if err := a.Connections().ValidateResp(resp, addr, "Unable to run command GETKEYSINSLOT"); err != nil {
			return allKeys, err
		}
		keys, err := resp.List()
		if err != nil {
			glog.Errorf("Wrong retured format for CLUSTER GETKEYSINSLOT: %v", err)
			return allKeys, err
		}

		allKeys = append(allKeys, keys...)

		keyCount += len(keys)
		if limit || len(keys) == 0 {
			break
		}
	}
	return allKeys, nil
}

// CountKeysInSlot exec the redis command to count the number of keys in the given slot on a node
func (a *Admin) CountKeysInSlot(addr string, slot Slot) (int64, error) {
	c, err := a.Connections().Get(addr)
	if err != nil {
		return 0, err
	}

	resp := c.Cmd("CLUSTER", "COUNTKEYSINSLOT", slot)
	if err := a.Connections().ValidateResp(resp, addr, "Unable to run command COUNTKEYSINSLOT"); err != nil {
		return 0, err
	}
	return resp.Int64()
}

// MigrateKeys use to migrate keys from slots to other slots. if replace is true, replace key on busy error
// timeout is in milliseconds
func (a *Admin) MigrateKeys(addr string, dest *Node, slots []Slot, batch int, timeout int, replace bool) (int, error) {
	if len(slots) == 0 {
		return 0, nil
	}
	keyCount := 0
	c, err := a.Connections().Get(addr)
	if err != nil {
		return keyCount, err
	}
	timeoutStr := strconv.Itoa(timeout)
	batchStr := strconv.Itoa(batch)

	for _, slot := range slots {
		for {
			resp := c.Cmd("CLUSTER", "GETKEYSINSLOT", slot, batchStr)
			if err := a.Connections().ValidateResp(resp, addr, "Unable to run command GETKEYSINSLOT"); err != nil {
				return keyCount, err
			}
			keys, err := resp.List()
			if err != nil {
				glog.Errorf("Wrong retured format for CLUSTER GETKEYSINSLOT: %v", err)
				return keyCount, err
			}

			keyCount += len(keys)
			if len(keys) == 0 {
				break
			}

			var args []string
			if replace {
				args = append([]string{dest.IP, dest.Port, "", "0", timeoutStr, "REPLACE", "KEYS"}, keys...)
			} else {
				args = append([]string{dest.IP, dest.Port, "", "0", timeoutStr, "KEYS"}, keys...)
			}

			resp = c.Cmd("MIGRATE", args)
			if err := a.Connections().ValidateResp(resp, addr, "Unable to run command MIGRATE"); err != nil {
				return keyCount, err
			}
		}
	}

	return keyCount, nil
}

// AttachSlaveToMaster attach a slave to a master node
func (a *Admin) AttachSlaveToMaster(slave *Node, master *Node) error {
	c, err := a.Connections().Get(slave.IPPort())
	if err != nil {
		return err
	}

	resp := c.Cmd("CLUSTER", "REPLICATE", master.ID)
	if err := a.Connections().ValidateResp(resp, slave.IPPort(), "Unable to run command REPLICATE"); err != nil {
		return err
	}

	slave.SetReferentMaster(master.ID)
	slave.SetRole(redisSlaveRole)

	return nil
}

// DetachSlave use to detach a slave to a master
func (a *Admin) DetachSlave(slave *Node) error {
	c, err := a.Connections().Get(slave.IPPort())
	if err != nil {
		glog.Errorf("unable to get the connection for slave ID:%s, addr:%s , err:%v", slave.ID, slave.IPPort(), err)
		return err
	}

	resp := c.Cmd("CLUSTER", "RESET", "SOFT")
	if err = a.Connections().ValidateResp(resp, slave.IPPort(), "Cannot attach node to cluster"); err != nil {
		return err
	}

	if err = a.AttachNodeToCluster(slave.IPPort()); err != nil {
		glog.Errorf("[DetachSlave] unable to AttachNodeToCluster the Slave id: %s addr:%s", slave.ID, slave.IPPort())
		return err
	}

	slave.SetReferentMaster("")
	slave.SetRole(redisMasterRole)

	return nil
}

// FlushAndReset flush the cluster and reset the cluster configuration of the node. Commands are piped, to ensure no items arrived between flush and reset
func (a *Admin) FlushAndReset(addr string, mode string) error {
	c, err := a.Connections().Get(addr)
	if err != nil {
		return err
	}
	c.PipeAppend("FLUSHALL")
	c.PipeAppend("CLUSTER", "RESET", mode)

	if !a.Connections().ValidatePipeResp(c, addr, "Cannot reset node") {
		return fmt.Errorf("Cannot reset node %s", addr)
	}

	return nil
}

// FlushAll flush all keys in cluster
func (a *Admin) FlushAll() {
	c, err := a.Connections().GetRandom()
	if err != nil {
		return
	}

	c.Cmd("FLUSHALL")
}

func selectMySlaves(me *Node, nodes Nodes) (Nodes, error) {
	return nodes.GetNodesByFunc(func(n *Node) bool {
		return n.MasterReferent == me.ID
	})
}

func (a *Admin) getInfos(c ClientInterface, addr string) (*NodeInfos, error) {
	resp := c.Cmd("CLUSTER", "NODES")
	if err := a.Connections().ValidateResp(resp, addr, "Unable to retrieve Node Info"); err != nil {
		return nil, err
	}

	var raw string
	var err error
	raw, err = resp.Str()

	if err != nil {
		return nil, fmt.Errorf("Wrong format from CLUSTER NODES: %v", err)
	}

	nodeInfos := DecodeNodeInfos(&raw, addr)

	if glog.V(3) {
		//Retrieve server info for debugging
		resp = c.Cmd("INFO", "SERVER")
		if err = a.Connections().ValidateResp(resp, addr, "Unable to retrieve Node Info"); err != nil {
			return nil, err
		}
		raw, err = resp.Str()
		if err != nil {
			return nil, fmt.Errorf("Wrong format from INFO SERVER: %v", err)
		}

		var serverStartTime time.Time
		serverStartTime, err = DecodeNodeStartTime(&raw)

		if err != nil {
			return nil, err
		}

		nodeInfos.Node.ServerStartTime = serverStartTime
	}

	return nodeInfos, nil
}

//RebuildConnectionMap rebuild the connection map according to the given addresse
func (a *Admin) RebuildConnectionMap(addrs []string, options *AdminOptions) {
	a.cnx.Reset()
	a.cnx = NewAdminConnections(addrs, options)
}
