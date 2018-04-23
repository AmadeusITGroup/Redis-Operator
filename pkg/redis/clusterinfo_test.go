package redis

import (
	"reflect"
	"testing"
)

func TestNodeDecodeRedisInfoInvalidInput(t *testing.T) {
	input := "07c37dfeb235213a872192d90877d0cd55635b91 127.0.0"

	nodeinfo := DecodeNodeInfos(&input, "")
	if len(nodeinfo.Friends) != 0 {
		t.Error("nodes should contains zero Node len:", len(nodeinfo.Friends))
	}

	if !reflect.DeepEqual(nodeinfo.Node, NewDefaultNode()) {
		t.Errorf("Shouldn't have any node, got: %s", nodeinfo.Node)
	}
}

func TestNodeDecodeRedisInfoOneSlave(t *testing.T) {
	input := "07c37dfeb235213a872192d90877d0cd55635b91 127.0.0.1:30004 myself,slave e7d1eecce10fd6bb5eb35b9f99a514335d9ba9ca 0 1426238317239 4 connected"

	nodeinfos := DecodeNodeInfos(&input, "")
	if nodeinfos.Node == nil {
		t.Error("nodeinfos should have one Node")
	}
	if len(nodeinfos.Friends) != 0 {
		t.Error("nodes should be alone in the cluster:", len(nodeinfos.Friends))
	}

	n := nodeinfos.Node
	if n.ID != "07c37dfeb235213a872192d90877d0cd55635b91" {
		t.Error("Wrong Node id")
	}
	if n.IPPort() != "127.0.0.1:30004" {
		t.Error("Wrong Node IPPort")
	}
	if n.Role != redisSlaveRole {
		t.Error("Wrong Node Role, should be slave")
	}

	if len(n.FailStatus) != 0 {
		t.Error("Wrong Node FailureStatus")
	}

	if n.MasterReferent != "e7d1eecce10fd6bb5eb35b9f99a514335d9ba9ca" {
		t.Error("Wrong Master Referent ")
	}

	if n.PingSent != 0 {
		t.Error("wrong PingSent value")
	}

	if n.PongRecv != 1426238317239 {
		t.Error("wrong PongRecv value")
	}

	if n.ConfigEpoch != 4 {
		t.Error("wrong ConfigEpoch value [badvalue]:", n.ConfigEpoch)
	}

	if n.LinkState != RedisLinkStateConnected {
		t.Error("wrong LinkState")
	}
}

func TestNodeDecodeRedisInfoOneMaster(t *testing.T) {
	input := "67ed2db8d677e59ec4a4cefb06858cf2a1a89fa1 127.0.0.1:30002 myself,master - 0 1426238316232 2 connected 5461-10922"
	nodeinfos := DecodeNodeInfos(&input, "")
	if nodeinfos.Node == nil {
		t.Error("nodeinfos should have one Node")
	}
	if len(nodeinfos.Friends) != 0 {
		t.Error("nodes should be alone in the cluster:", len(nodeinfos.Friends))
	}

	n := nodeinfos.Node
	if n.Role != redisMasterRole {
		t.Error("Wrong Node Role, should be slave")
	}

	if len(n.Slots) != (10922 - 5461 + 1) {
		t.Errorf("Master should have %d slots", 10922-5461+1)
	}
}

func TestNodeDecodeRedisInfoMultiNodes(t *testing.T) {
	input := `07c37dfeb235213a872192d90877d0cd55635b91 127.0.0.1:30004 slave e7d1eecce10fd6bb5eb35b9f99a514335d9ba9ca 0 1426238317239 4 connected
67ed2db8d677e59ec4a4cefb06858cf2a1a89fa1 127.0.0.1:30002 master - 0 1426238316232 2 connected 5461-10922
292f8b365bb7edb5e285caf0b7e6ddc7265d2f4f 127.0.0.1:30003 master - 0 1426238318243 3 connected 10923-16383
6ec23923021cf3ffec47632106199cb7f496ce01 127.0.0.1:30005 slave 67ed2db8d677e59ec4a4cefb06858cf2a1a89fa1 0 1426238316232 5 connected
824fe116063bc5fcf9f4ffd895bc17aee7731ac3 127.0.0.1:30006 slave 292f8b365bb7edb5e285caf0b7e6ddc7265d2f4f 0 1426238317741 6 connected
e7d1eecce10fd6bb5eb35b9f99a514335d9ba9ca :30001 myself,master - 0 0 1 connected 0-5460`

	nodeinfos := DecodeNodeInfos(&input, "127.0.0.1")
	if len(nodeinfos.Friends) != 5 {
		t.Error("nodeinfo friends should contains 5 Node len:", len(nodeinfos.Friends))
	}
}

func TestNodeToString(t *testing.T) {
	input := "67ed2db8d677e59ec4a4cefb06858cf2a1a89fa1 127.0.0.1:30002 myself,master - 0 1426238316232 2 connected 5461-5471"
	nodeinfos := DecodeNodeInfos(&input, "")
	if len(nodeinfos.Friends) != 0 {
		t.Error("nodes should be alone in the cluster:", len(nodeinfos.Friends))
	} else {

		output :=
			`{Redis ID: 67ed2db8d677e59ec4a4cefb06858cf2a1a89fa1, role: Master, master: , link: connected, status: [], addr: 127.0.0.1:30002, slots: [5461-5471], len(migratingSlots): 0, len(importingSlots): 0}`

		if nodeinfos.Node.String() != output {
			t.Errorf("String output not valide: expected '%s', got '%s'", output, nodeinfos.Node.String())
		}
	}

}
