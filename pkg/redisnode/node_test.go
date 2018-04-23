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
		RedisServerPort: "1234",
		RedisMaxMemory:  0,
		Redis: config.Redis{
			ClusterNodeTimeout: 321,
			ConfigFile:         configfile.Name(),
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
	expected := "port 1234\nbind " + node.IP + " 127.0.0.1\ncluster-node-timeout 321\n"
	if expected != string(content) {
		t.Errorf("Wrong file content, expected '%s', got '%s'", expected, string(content))
	}
}

func TestAdminCommands(t *testing.T) {
	a := admin.NewFakeAdmin([]string{})
	c := Config{
		RedisServerPort: "1234",
	}

	node := NewNode(&c, a)
	defer node.Clear()

	// all methods below simply call the fake admin, test currently only improves coverage
	node.InitRedisCluster("1.1.1.1")
	node.AttachNodeToCluster("1.1.1.1")
	node.ForgetNode()
	node.StartFailover()
}
