package redisnode

import (
	"errors"
	"io/ioutil"
	//	"net"
	"os"
	"testing"

	kapi "k8s.io/api/core/v1"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kfakeclient "k8s.io/client-go/kubernetes/fake"

	"github.com/amadeusitgroup/redis-operator/pkg/config"
	"github.com/amadeusitgroup/redis-operator/pkg/redis/fake"
	"github.com/amadeusitgroup/redis-operator/pkg/redis/fake/admin"
)

func TestTestAndWaitConnection(t *testing.T) {
	redisSrv1 := fake.NewRedisServer(t)
	defer redisSrv1.Close()
	addr1 := redisSrv1.GetHostPort()

	rq := "PING"
	resp := "PONG"
	redisSrv1.PushResponse(rq, resp)

	err := testAndWaitConnection(addr1, 1)
	if err != nil {
		t.Errorf("Unexpected error while waiting for fake redis node: %v", err)
	}
}

func TestIsClusterInitialization(t *testing.T) {
	currentIP := "1.2.3.4"
	conf := Config{
		Redis:   config.Redis{ServerPort: "1234"},
		Cluster: config.Cluster{Namespace: "default", NodeService: "redis-service"},
	}

	testCases := []struct {
		name             string
		endpoints        kapi.Endpoints
		isInitialization bool
	}{
		{
			name:             "1) test init true",
			endpoints:        kapi.Endpoints{ObjectMeta: kmetav1.ObjectMeta{Name: "redis-service", Namespace: "default"}, Subsets: []kapi.EndpointSubset{}}, //empty
			isInitialization: true,
		},
		{
			name: "2) test init false",
			endpoints: kapi.Endpoints{ObjectMeta: kmetav1.ObjectMeta{Name: "redis-service", Namespace: "default"}, Subsets: []kapi.EndpointSubset{
				{Addresses: []kapi.EndpointAddress{{IP: "1.0.0.1"}, {IP: "1.0.0.2"}, {IP: "1.0.0.3"}}, Ports: []kapi.EndpointPort{{Port: 1234}}}, // full
			}},
			isInitialization: false,
		},
		{
			name: "3) test init true",
			endpoints: kapi.Endpoints{ObjectMeta: kmetav1.ObjectMeta{Name: "redis-service", Namespace: "default"}, Subsets: []kapi.EndpointSubset{
				{Addresses: []kapi.EndpointAddress{{IP: currentIP}}, Ports: []kapi.EndpointPort{{Port: 1234}}}, // only current
			}},
			isInitialization: true,
		},
	}

	for _, tc := range testCases {
		node := &RedisNode{
			config:     &conf,
			kubeClient: kfakeclient.NewSimpleClientset(&tc.endpoints),
		}
		_, isInit := node.isClusterInitialization(currentIP)
		if isInit != tc.isInitialization {
			t.Errorf("[case: %s] Wrong init status returned, expected %t, got %t", tc.name, tc.isInitialization, isInit)
		}
	}
}

func TestRedisInitializationAttach(t *testing.T) {
	myIP := "1.2.3.4"

	tmpfile, err := ioutil.TempFile("", "config")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	c := &Config{
		Redis:   config.Redis{ServerPort: "1234", ConfigFileName: tmpfile.Name()},
		Cluster: config.Cluster{Namespace: "default", NodeService: "redis-service"},
	}

	// other ips registered, will attach to them
	fakeAdmin := admin.NewFakeAdmin([]string{myIP})
	fakeAdmin.InitRedisClusterRet[myIP] = errors.New("Should not call init cluster")

	endpoint := kapi.Endpoints{
		ObjectMeta: kmetav1.ObjectMeta{
			Name:      "redis-service",
			Namespace: "default",
		},
		Subsets: []kapi.EndpointSubset{
			{
				Addresses: []kapi.EndpointAddress{
					{IP: "1.1.1.1"},
					{IP: "2.2.2.2"},
				},
			},
		},
	}

	rn := &RedisNode{
		config:     c,
		redisAdmin: fakeAdmin,
		kubeClient: kfakeclient.NewSimpleClientset(&endpoint),
	}

	node, err := rn.init()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if node == nil {
		t.Error("node should not be nil")
	}
}
