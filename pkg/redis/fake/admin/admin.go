package admin

import (
	"github.com/amadeusitgroup/redis-operator/pkg/redis"
)

// MigrateKeyRetType structure to describe the return data of the MigrateKey method
type MigrateKeyRetType struct {
	Nb  int
	Err error
}

// GetClusterInfoRetType structure to describe the return data of GetClusterInfo method
type GetClusterInfoRetType struct {
	Nodes redis.Nodes
	Err   error
}

// GetKeysInSlotRetType structure to describe the return data of GetKeysInSlot method
type GetKeysInSlotRetType struct {
	Keys []string
	Err  error
}

// CountKeysInSlotRetType structure to describe the return data of CountKeysInSlot method
type CountKeysInSlotRetType struct {
	NbKeys int64
	Err    error
}

// ClusterInfosRetType structure to describe the return data of GetClusterInfosRet method
type ClusterInfosRetType struct {
	ClusterInfos *redis.ClusterInfos
	Err          error
}

// Admin representation of a fake redis admin, where all
// return error for fake admin commands are configurable per addr, and return nil by default
type Admin struct {
	// HashMaxSlots Max slot value
	HashMaxSlots redis.Slot
	// InitRedisClusterRet map of returned error for InitRedisCluster function
	InitRedisClusterRet map[string]error
	// GetClusterInfosRet returned value for GetClusterInfos function
	GetClusterInfosRet ClusterInfosRetType
	// GetClusterInfosSelectedRet returned value for GetClusterInfos function
	GetClusterInfosSelectedRet ClusterInfosRetType
	// AttachNodeToClusterRet map of returned error for AttachNodeToCluster function
	AttachNodeToClusterRet map[string]error
	// UpdateClientsRet map of returned error for UpdateClients function
	UpdateClientsRet map[string]error
	// StartFailoverRet map of returned error for StartFailover function
	StartFailoverRet map[string]error
	// ForgetNodeRet map of returned error for ForgetNode function
	ForgetNodeRet map[string]error
	// SetSlotsRet map of returned error for SetSlots function
	SetSlotsRet map[string]error
	// AddSlotsRet map of returned error for AddSlots function
	AddSlotsRet map[string]error
	// DelSlotsRet map of returned error for DellSlots function
	DelSlotsRet map[string]error
	// GetKeysInSlotRet map of returned data for GetKeysInSlot function
	GetKeysInSlotRet map[string]GetKeysInSlotRetType
	// CountKeysInSlotRet map of returned data for for CountKeysInSlot function
	CountKeysInSlotRet map[string]CountKeysInSlotRetType
	// MigrateKeysRet map of returned error for MigrateKeys function
	MigrateKeysRet map[string]MigrateKeyRetType
	// AttachSlaveToMasterRet map of returned error for AttachSlaveToMaster function
	AttachSlaveToMasterRet map[string]error
	// DetachSlaveToMasterRet map of returned error for DetachSlave function
	DetachSlaveToMasterRet map[string]error
	// ResetRet map of returned error for FlushAndReset function
	FlushAndResetRet map[string]error
	cnx              *Connections
}

// NewFakeAdmin returns new AdminInterface for fake admin
func NewFakeAdmin(addrs []string) *Admin {
	return &Admin{
		HashMaxSlots:               16383,
		InitRedisClusterRet:        make(map[string]error),
		GetClusterInfosRet:         ClusterInfosRetType{},
		GetClusterInfosSelectedRet: ClusterInfosRetType{},
		AttachNodeToClusterRet:     make(map[string]error),
		UpdateClientsRet:           make(map[string]error),
		StartFailoverRet:           make(map[string]error),
		ForgetNodeRet:              make(map[string]error),
		SetSlotsRet:                make(map[string]error),
		AddSlotsRet:                make(map[string]error),
		DelSlotsRet:                make(map[string]error),
		GetKeysInSlotRet:           make(map[string]GetKeysInSlotRetType),
		CountKeysInSlotRet:         make(map[string]CountKeysInSlotRetType),
		MigrateKeysRet:             make(map[string]MigrateKeyRetType),
		AttachSlaveToMasterRet:     make(map[string]error),
		DetachSlaveToMasterRet:     make(map[string]error),
		FlushAndResetRet:           make(map[string]error),
		cnx:                        &Connections{},
	}
}

// Close used to close all possible resources instanciate by the Admin
func (a *Admin) Close() {
}

// Connections returns a connection map
func (a *Admin) Connections() redis.AdminConnectionsInterface {
	return a.cnx
}

// GetHashMaxSlot get the max slot value
func (a *Admin) GetHashMaxSlot() redis.Slot {
	return a.HashMaxSlots
}

// AttachNodeToCluster command use to connect a Node to the cluster
func (a *Admin) AttachNodeToCluster(addr string) error {
	val, ok := a.AttachNodeToClusterRet[addr]
	if !ok {
		val = nil
	}
	return val
}

// InitRedisCluster used to init a single node redis cluster
func (a *Admin) InitRedisCluster(addr string) error {
	val, ok := a.InitRedisClusterRet[addr]
	if !ok {
		val = nil
	}
	return val
}

// GetClusterInfos returns redis cluster infos from all clients
func (a *Admin) GetClusterInfos() (*redis.ClusterInfos, error) {
	return a.GetClusterInfosRet.ClusterInfos, a.GetClusterInfosRet.Err
}

// GetClusterInfosSelected returns redis cluster infos from all clients
func (a *Admin) GetClusterInfosSelected(addr []string) (*redis.ClusterInfos, error) {
	return a.GetClusterInfosSelectedRet.ClusterInfos, a.GetClusterInfosSelectedRet.Err
}

// StartFailover used to force the failover of a specific redis master node
func (a *Admin) StartFailover(addr string) error {
	val, ok := a.StartFailoverRet[addr]
	if !ok {
		val = nil
	}
	return val
}

// ForgetNode used to force other redis cluster node to forget a specific node
func (a *Admin) ForgetNode(addr string) error {
	val, ok := a.ForgetNodeRet[addr]
	if !ok {
		val = nil
	}
	return val
}

// ForgetNodeByAddr used to force other redis cluster node to forget a specific node
func (a *Admin) ForgetNodeByAddr(addr string) error {
	val, ok := a.ForgetNodeRet[addr]
	if !ok {
		val = nil
	}
	return val
}

// SetSlots use to set SETSLOT command on several slots
func (a *Admin) SetSlots(addr, action string, slots []redis.Slot, nodeID string) error {
	val, ok := a.SetSlotsRet[addr]
	if !ok {
		val = nil
	}
	return val
}

// AddSlots use to set ADDSLOT command on several slots
func (a *Admin) AddSlots(addr string, slots []redis.Slot) error {
	val, ok := a.AddSlotsRet[addr]
	if !ok {
		val = nil
	}
	return val
}

// DelSlots exec the redis command to del slots in a pipeline
func (a *Admin) DelSlots(addr string, slots []redis.Slot) error {
	val, ok := a.DelSlotsRet[addr]
	if !ok {
		val = nil
	}
	return val
}

// GetKeysInSlot exec the redis command to get the keys in the given slot on the node we are connected to
func (a *Admin) GetKeysInSlot(addr string, node redis.Slot, batch int, limit bool) ([]string, error) {
	val, ok := a.GetKeysInSlotRet[addr]
	if !ok {
		val = GetKeysInSlotRetType{Keys: []string{}, Err: nil}
	}
	return val.Keys, val.Err
}

// CountKeysInSlot exec the redis command to count the keys in the given slot of the node
func (a *Admin) CountKeysInSlot(addr string, node redis.Slot) (int64, error) {
	val, ok := a.CountKeysInSlotRet[addr]
	if !ok {
		val = CountKeysInSlotRetType{NbKeys: int64(0), Err: nil}
	}
	return val.NbKeys, val.Err
}

// MigrateKeys use to migrate keys from slots to other slots
func (a *Admin) MigrateKeys(addr string, dest *redis.Node, slots []redis.Slot, batch, timeout int, replace bool) (int, error) {
	val, ok := a.MigrateKeysRet[addr]
	if !ok {
		val = MigrateKeyRetType{Nb: 0, Err: nil}
	}
	return val.Nb, val.Err
}

// AttachSlaveToMaster attach a slave to a master node
func (a *Admin) AttachSlaveToMaster(slave *redis.Node, master *redis.Node) error {
	val, ok := a.AttachSlaveToMasterRet[master.ID]
	if !ok {
		val = nil
	}
	return val
}

// DetachSlave dettach a slave to its master
func (a *Admin) DetachSlave(slave *redis.Node) error {
	val, ok := a.DetachSlaveToMasterRet[slave.ID]
	if !ok {
		val = nil
	}
	return val
}

// FlushAndReset reset the cluster configuration of the node, also flush in the same pipe
func (a *Admin) FlushAndReset(addr string, mode string) error {
	val, ok := a.FlushAndResetRet[addr]
	if !ok {
		val = nil
	}
	return val
}

// FlushAll flush all keys in cluster
func (a *Admin) FlushAll() {
}

//RebuildConnectionMap rebuild the connection map according to the given addresse
func (a *Admin) RebuildConnectionMap(addrs []string, options *redis.AdminOptions) {
}
