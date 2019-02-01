package redisnode

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/amadeusitgroup/redis-operator/pkg/config"
	"github.com/amadeusitgroup/redis-operator/pkg/redis/fake/admin"
)

func TestUpdateNodeConfigFile(t *testing.T) {
	temp, _ := ioutil.TempDir("", "test")
	configfile, createerr := os.Create(filepath.Join(temp, "redisconfig.conf"))
	if createerr != nil {
		t.Errorf("Couldn' t create temporary config file: %v", createerr)
	}
	defer os.RemoveAll(temp)
	configfile.Close()

	a := admin.NewFakeAdmin([]string{})
	c := Config{
		Redis: config.Redis{
			ServerPort:         "1234",
			MaxMemory:          1048576,
			MaxMemoryPolicy:    "allkeys-lru",
			ClusterNodeTimeout: 321,
			ConfigFileName:     configfile.Name(),
			ConfigFiles:        []string{"/cfg/foo.cfg", "bar.cfg"},
		},
	}

	node := NewNode(&c, a)
	defer node.Clear()
	err := node.UpdateNodeConfigFile()
	if err != nil {
		t.Errorf("Unexpected error while updating config file: %v", err)
	}

	// checking file content
	content, _ := ioutil.ReadFile(configfile.Name())
	var expected = `include /redis-conf/redis.conf
port 1234
cluster-enabled yes
maxmemory 1048576
maxmemory-policy allkeys-lru
bind 0.0.0.0
cluster-config-file /redis-data/node.conf
dir /redis-data
cluster-node-timeout 321
include /cfg/foo.cfg
include bar.cfg
`
	if expected != string(content) {
		t.Errorf("Wrong file content, expected '%s', got '%s'", expected, string(content))
	}
}

func TestAdminCommands(t *testing.T) {
	a := admin.NewFakeAdmin([]string{})
	c := Config{
		Redis: config.Redis{ServerPort: "1234"},
	}

	node := NewNode(&c, a)
	defer node.Clear()

	// all methods below simply call the fake admin, test currently only improves coverage
	node.InitRedisCluster("1.1.1.1")
	node.AttachNodeToCluster("1.1.1.1")
	node.ForgetNode()
	node.StartFailover()
}
