// +build !race

package sanitycheck

import (
	"fmt"
	"net"
	"reflect"
	"testing"

	//kapi "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rapi "github.com/amadeusitgroup/redis-operator/pkg/api/redis/v1"
	"github.com/amadeusitgroup/redis-operator/pkg/config"
	"github.com/amadeusitgroup/redis-operator/pkg/redis"
	"github.com/amadeusitgroup/redis-operator/pkg/redis/fake"
)

func TestFixClusterSplit(t *testing.T) {

	// const objects
	redisSrv1 := fake.NewRedisServer(t)
	defer redisSrv1.Close()
	addr1 := redisSrv1.GetHostPort()
	redisSrv2 := fake.NewRedisServer(t)
	defer redisSrv2.Close()
	addr2 := redisSrv2.GetHostPort()
	redisSrv3 := fake.NewRedisServer(t)
	defer redisSrv3.Close()
	addr3 := redisSrv3.GetHostPort()
	host3, port3, _ := net.SplitHostPort(addr3)

	admin := redis.NewAdmin([]string{addr1, addr2, addr3}, nil)
	cfg := &config.Redis{}
	redisNodeID1 := "07c37dfeb235213a872192d90877d0cd55635b91"
	redisNodeID2 := "7d1eecce10fd6bb5eb35b9f99a514335d9ba9ca"
	redisNodeID3 := "824fe116063bc5fcf9f4ffd895bc17aee7731ac3"

	// setup answers to redis queries
	rqnodes := "CLUSTER NODES"
	resp1 := fmt.Sprintf(`%s %s myself,master - 0 0 1 connected 0 %d
	%s %s slave %s 0 1426238317239 2 connected`, redisNodeID1, addr1, admin.GetHashMaxSlot(), redisNodeID2, addr2, redisNodeID1)
	resp2 := fmt.Sprintf(`%s %s myself,slave %s 0 1426238317239 2 connected
	%s %s master - 0 0 1 connected 0 %d`, redisNodeID2, addr2, redisNodeID1, redisNodeID1, addr1, admin.GetHashMaxSlot())
	resp3 := fmt.Sprintf(`%s %s myself,master - 0 0 1 connected 0 %d`, redisNodeID3, addr3, admin.GetHashMaxSlot())

	rqMeet3 := fmt.Sprintf("CLUSTER MEET %s %s", host3, port3)

	//push twice because a random node will receive the request twice
	redisSrv1.PushResponse(rqnodes, resp1)
	redisSrv1.PushResponse(rqnodes, resp1)
	redisSrv2.PushResponse(rqnodes, resp2)
	redisSrv2.PushResponse(rqnodes, resp2)
	redisSrv3.PushResponse(rqnodes, resp3)
	redisSrv3.PushResponse(rqnodes, resp3)

	redisSrv3.PushResponse("FLUSHALL", "OK")
	redisSrv3.PushResponse("CLUSTER RESET HARD", "OK")
	redisSrv1.PushResponse(rqMeet3, "OK")
	redisSrv2.PushResponse(rqMeet3, "OK")

	for key, val := range redisSrv3.Responses {
		t.Logf("response: [%s] [%v]", key, val)
	}

	infos, err := admin.GetClusterInfos()
	if err == nil {
		t.Errorf("admin.GetClusterInfos() should return an error")
	}

	// First run, should return an inconsitent error
	if action, err := FixClusterSplit(admin, cfg, infos, false); err != nil && action {
		t.Errorf("FixClusterSplit should not return an error and action==true. action[%v] error[%v]", action, err)
	}
}

func TestBuildClustersLists(t *testing.T) {
	// In the test below, we cannot directly use initialize redis.NodeSlice in redis.NodeInfos, this is a go vet issue: https://github.com/golang/go/issues/9171
	ip1 := redis.Nodes{{IP: "ip1", Port: "1234"}}
	ip2 := redis.Nodes{{IP: "ip2", Port: "1234"}}
	ip56 := redis.Nodes{{IP: "ip5", Port: "1234"}, {IP: "ip6", Port: "1234"}}
	ip64 := redis.Nodes{{IP: "ip6", Port: "1234"}, {IP: "ip4", Port: "1234"}}
	ip54 := redis.Nodes{{IP: "ip5", Port: "1234"}, {IP: "ip4", Port: "1234"}}
	// end of workaround
	testCases := []struct {
		input  *redis.ClusterInfos
		output []cluster
	}{ //several partilly different cannot happen, so not tested
		{ // empty
			input:  &redis.ClusterInfos{Infos: map[string]*redis.NodeInfos{}, Status: redis.ClusterInfosConsistent},
			output: []cluster{},
		},
		{ // one node
			input:  &redis.ClusterInfos{Infos: map[string]*redis.NodeInfos{"ip1:1234": {Node: &redis.Node{IP: "ip1", Port: "1234"}, Friends: redis.Nodes{}}}, Status: redis.ClusterInfosConsistent},
			output: []cluster{{"ip1:1234"}},
		},
		{ // no discrepency
			input: &redis.ClusterInfos{
				Infos: map[string]*redis.NodeInfos{
					"ip1:1234": {Node: &redis.Node{IP: "ip1", Port: "1234"}, Friends: ip2},
					"ip2:1234": {Node: &redis.Node{IP: "ip2", Port: "1234"}, Friends: ip1},
				},
				Status: redis.ClusterInfosConsistent,
			},
			output: []cluster{{"ip1:1234", "ip2:1234"}},
		},
		{ // several decorelated
			input: &redis.ClusterInfos{
				Infos: map[string]*redis.NodeInfos{
					"ip1:1234": {Node: &redis.Node{IP: "ip1", Port: "1234"}, Friends: ip2},
					"ip2:1234": {Node: &redis.Node{IP: "ip2", Port: "1234"}, Friends: ip1},
					"ip3:1234": {Node: &redis.Node{IP: "ip3", Port: "1234"}, Friends: redis.Nodes{}},
					"ip4:1234": {Node: &redis.Node{IP: "ip4", Port: "1234"}, Friends: ip56},
					"ip5:1234": {Node: &redis.Node{IP: "ip5", Port: "1234"}, Friends: ip64},
					"ip6:1234": {Node: &redis.Node{IP: "ip6", Port: "1234"}, Friends: ip54},
				},
				Status: redis.ClusterInfosInconsistent,
			},
			output: []cluster{{"ip1:1234", "ip2:1234"}, {"ip3:1234"}, {"ip4:1234", "ip5:1234", "ip6:1234"}},
		},
		{ // empty ignored
			input: &redis.ClusterInfos{
				Infos: map[string]*redis.NodeInfos{
					"ip1:1234": {Node: &redis.Node{IP: "ip1", Port: "1234"}, Friends: ip2},
					"ip2:1234": {Node: &redis.Node{IP: "ip2", Port: "1234"}, Friends: ip1},
					"ip3:1234": nil,
				},
				Status: redis.ClusterInfosInconsistent,
			},
			output: []cluster{{"ip1:1234", "ip2:1234"}},
		},
	}

	for i, tc := range testCases {
		output := buildClustersLists(tc.input)
		// because we work with map, order might not be conserved
		if !compareClusters(output, tc.output) {
			t.Errorf("[Case %d] Unexpected result for buildClustersLists, expected %v, got %v", i, tc.output, output)
		}
	}
}

func TestSplitMainCluster(t *testing.T) {
	testCases := []struct {
		inputClusters []cluster
		keptCluster   cluster
		toFixClusters []cluster
	}{
		{[]cluster{}, cluster{}, []cluster{}},                                         // empty
		{[]cluster{{}}, cluster{}, []cluster{}},                                       // empty in a strange way
		{[]cluster{{"ip1"}}, cluster{"ip1"}, []cluster{}},                             // one node
		{[]cluster{{"ip1", "ip2", "ip3"}}, cluster{"ip1", "ip2", "ip3"}, []cluster{}}, // one cluster
		{[]cluster{{"ip1", "ip2", "ip3"}, {"ip4", "ip5"}, {"ip6"}, {"ip7"}},
			cluster{"ip1", "ip2", "ip3"}, []cluster{{"ip4", "ip5"}, {"ip6"}, {"ip7"}}}, // one big and several smalls first pos
		{[]cluster{{"ip1", "ip2"}, {"ip3", "ip4", "ip5"}, {"ip6"}, {"ip7"}},
			cluster{"ip3", "ip4", "ip5"}, []cluster{{"ip1", "ip2"}, {"ip6"}, {"ip7"}}}, // one big and several smalls middle
		{[]cluster{{"ip1", "ip2"}, {"ip3", "ip4"}, {"ip5"}, {"ip6", "ip7", "ip8"}},
			cluster{"ip6", "ip7", "ip8"}, []cluster{{"ip1", "ip2"}, {"ip3", "ip4"}, {"ip5"}}}, // one big and several smalls end pos
		{[]cluster{{"ip1", "ip2"}, {"ip3", "ip4"}, {"ip5", "ip6"}, {"ip7"}},
			cluster{"ip1", "ip2"}, []cluster{{"ip3", "ip4"}, {"ip5", "ip6"}, {"ip7"}}}, // equal size
		{[]cluster{{"ip1"}, {"ip2"}, {"ip3"}}, cluster{"ip1"}, []cluster{{"ip2"}, {"ip3"}}}, // all equal to one
	}

	for i, tc := range testCases {
		main, other := splitMainCluster(tc.inputClusters)
		if !reflect.DeepEqual(main, tc.keptCluster) {
			t.Errorf("[Case %d] Unexpected result for main cluster, expected %v, got %v", i, tc.keptCluster, main)
		}
		if !reflect.DeepEqual(other, tc.toFixClusters) {
			t.Errorf("[Case %d] Unexpected result for to fix clusters, expected %v, got %v", i, tc.toFixClusters, other)
		}
	}
}

// newCluster generate a new Cluster struct
func newCluster(replicaFactor int32, nbMaster int32) *rapi.RedisCluster {
	return &rapi.RedisCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "",
		},
		Spec: rapi.RedisClusterSpec{
			NumberOfMaster:    &nbMaster,
			ReplicationFactor: &replicaFactor,
		},
	}
}

func compareClusters(c1, c2 []cluster) bool {
	if len(c1) != len(c2) {
		return false
	}

	for _, c1elem := range c2 {
		found := false
		for _, c2elem := range c1 {
			if compareCluster(c1elem, c2elem) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func compareCluster(c1, c2 cluster) bool {
	if len(c1) != len(c2) {
		return false
	}
	for _, c1elem := range c2 {
		found := false
		for _, c2elem := range c1 {
			if c1elem == c2elem {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}
