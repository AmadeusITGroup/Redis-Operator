package redisnode

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"

	"github.com/amadeusitgroup/redis-operator/pkg/config"
	"github.com/amadeusitgroup/redis-operator/pkg/redis"
	"github.com/amadeusitgroup/redis-operator/pkg/utils"
	"github.com/golang/glog"
)

const (
	dataFolder = "/redis-data"
)

// Node struct that represent a RedisNodeWrapper
type Node struct {
	config     *Config
	IP         string
	Addr       string
	RedisAdmin redis.AdminInterface
}

// NewNode return a instance of a Node
func NewNode(c *Config, admin redis.AdminInterface) *Node {
	ip, err := utils.GetMyIP()
	if err != nil {
		return nil
	}

	n := &Node{
		config:     c,
		RedisAdmin: admin,
		IP:         ip,
		Addr:       net.JoinHostPort(ip, c.Redis.ServerPort),
	}

	return n
}

// Clear clear possible initialize ressource
func (n *Node) Clear() {

	if n.RedisAdmin != nil {
		n.RedisAdmin.Close()
	}

}

// UpdateNodeConfigFile update the redis config file with node information: ip, port
func (n *Node) UpdateNodeConfigFile() error {
	if n.config.Redis.ConfigFileName != config.RedisConfigFileDefault {
		if err := n.addSettingInConfigFile("include " + config.RedisConfigFileDefault); err != nil {
			return err
		}
	}

	if err := n.addSettingInConfigFile("port " + n.config.Redis.ServerPort); err != nil {
		return err
	}

	if err := n.addSettingInConfigFile("cluster-enabled yes"); err != nil {
		return err
	}

	if n.config.Redis.MaxMemory > 0 {
		if err := n.addSettingInConfigFile(fmt.Sprintf("maxmemory %d", n.config.Redis.MaxMemory)); err != nil {
			return err
		}
	}
	if n.config.Redis.MaxMemoryPolicy != config.RedisMaxMemoryPolicyDefault {
		if err := n.addSettingInConfigFile(fmt.Sprintf("maxmemory-policy %s", n.config.Redis.MaxMemoryPolicy)); err != nil {
			return err
		}
	}

	if err := n.addSettingInConfigFile("bind " + n.IP + " 127.0.0.1"); err != nil {
		return err
	}

	if err := n.addSettingInConfigFile("cluster-config-file /redis-data/node.conf"); err != nil {
		return err
	}

	if err := n.addSettingInConfigFile("dir /redis-data"); err != nil {
		return err
	}

	if err := n.addSettingInConfigFile("cluster-node-timeout " + strconv.Itoa(n.config.Redis.ClusterNodeTimeout)); err != nil {
		return err
	}
	if n.config.Redis.GetRenameCommandsFile() != "" {

		if err := n.addSettingInConfigFile("include " + n.config.Redis.GetRenameCommandsFile()); err != nil {
			return err
		}
	}

	// Add at the end any configuration file provided as redis-node's arguments.
	for _, file := range n.config.Redis.ConfigFiles {
		if err := n.addSettingInConfigFile("include " + file); err != nil {
			return err
		}
	}

	return nil
}

// addSettingInConfigFile add a line in the redis configuration file
func (n *Node) addSettingInConfigFile(line string) error {
	f, err := os.OpenFile(n.config.Redis.ConfigFileName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("unable to set '%s' in config file, openfile error %s err:%v", line, n.config.Redis.ConfigFileName, err)
	}

	defer f.Close()

	_, err = f.WriteString(line + "\n")
	if err != nil {
		return fmt.Errorf("unable to set '%s' in config file, err:%v", line, err)
	}
	return nil
}

// InitRedisCluster used to init a redis cluster with the current node
func (n *Node) InitRedisCluster(addr string) error {
	glog.Info("InitRedis Cluster... starting")
	err := n.RedisAdmin.InitRedisCluster(addr)
	glog.Info("InitRedis Cluster... done")

	return err
}

// AttachNodeToCluster used to attach the current node to a redis cluster
func (n *Node) AttachNodeToCluster(addr string) error {
	glog.Info("AttachNodeToCluster... starting")

	return n.RedisAdmin.AttachNodeToCluster(addr)
}

// ForgetNode used to remove a node for a cluster
func (n *Node) ForgetNode() error {
	glog.Info("ForgetNode... starting")

	return n.RedisAdmin.ForgetNodeByAddr(n.Addr)
}

// StartFailover start Failover if needed
func (n *Node) StartFailover() error {
	glog.Info("StartFailover... starting")

	return n.RedisAdmin.StartFailover(n.Addr)
}

// ClearDataFolder completely erase all files in the /data folder
func (n *Node) ClearDataFolder() error {
	return clearFolder(dataFolder)
}

// ClearFolder remover all files and folder in a given folder
func clearFolder(folder string) error {
	glog.Infof("Clearing '%s' folder... ", folder)
	d, err := os.Open(folder)
	if err != nil {
		glog.Infof("Cannot access folder %s: %v", folder, err)
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		glog.Infof("Cannot read files in %s: %v", folder, err)
		return err
	}
	for _, name := range names {
		file := filepath.Join(folder, name)
		glog.V(2).Infof("Removing %s", file)
		err = os.RemoveAll(file)
		if err != nil {
			glog.Errorf("Error while removing %s: %v", file, err)
			return err
		}
	}
	return nil
}
