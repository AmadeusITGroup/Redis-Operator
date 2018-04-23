package config

import (
	"fmt"
	"path"

	"github.com/spf13/pflag"
)

const (
	// DefaultRedisTimeout default redis timeout (ms)
	DefaultRedisTimeout = 2000
	//DefaultClusterNodeTimeout default cluster node timeout (ms)
	//The maximum amount of time a Redis Cluster node can be unavailable, without it being considered as failing
	DefaultClusterNodeTimeout = 2000
	// RedisRenameCommandsDefaultPath default path to volume storing rename commands
	RedisRenameCommandsDefaultPath = "/etc/secret-volume"
	// RedisRenameCommandsDefaultFile default file name containing rename commands
	RedisRenameCommandsDefaultFile = ""
	// RedisConfigFileDefault default config file path
	RedisConfigFileDefault = "/redis-server/redis.conf"
)

// Redis used to store all Redis configuration information
type Redis struct {
	DialTimeout        int
	ClusterNodeTimeout int
	ConfigFile         string
	renameCommandsPath string
	renameCommandsFile string
}

// AddFlags use to add the Redis Config flags to the command line
func (r *Redis) AddFlags(fs *pflag.FlagSet) {
	fs.IntVar(&r.DialTimeout, "rdt", DefaultRedisTimeout, "redis dial timeout (ms)")
	fs.IntVar(&r.ClusterNodeTimeout, "cluster-node-timeout", DefaultClusterNodeTimeout, "redis node timeout (ms)")
	fs.StringVar(&r.ConfigFile, "c", RedisConfigFileDefault, "redis config file path")
	fs.StringVar(&r.renameCommandsPath, "rename-command-path", RedisRenameCommandsDefaultPath, "Path to the folder where rename-commands option for redis are available")
	fs.StringVar(&r.renameCommandsFile, "rename-command-file", RedisRenameCommandsDefaultFile, "Name of the file where rename-commands option for redis are available, disabled if empty")

}

// GetRenameCommandsFile return the path to the rename command file, or empty string if not define
func (r *Redis) GetRenameCommandsFile() string {
	if r.renameCommandsFile == "" {
		return ""
	}
	return path.Join(r.renameCommandsPath, r.renameCommandsFile)
}

// String stringer interface
func (r Redis) String() string {
	var output string
	output += fmt.Sprintln("[ Redis Configuration ]")
	output += fmt.Sprintln("- DialTimeout:", r.DialTimeout)
	output += fmt.Sprintln("- ClusterNodeTimeout:", r.ClusterNodeTimeout)
	output += fmt.Sprintln("- Rename commands:", r.GetRenameCommandsFile())
	return output
}
