package initcontainer

import (
	"path"

	"github.com/spf13/pflag"
)

const (
	// RedisConfigPortDefault default redis port
	RedisConfigPortDefault = "6379"
	// RedisConfigFileDefault default config file path
	RedisConfigFileDefault = "/redis-server/redis.conf"
	// RedisRenameCommandsDefaultPath default path to volume storing rename commands
	RedisRenameCommandsDefaultPath = "/etc/secret-volume"
	// RedisRenameCommandsDefaultFile default file name containing rename commands
	RedisRenameCommandsDefaultFile = ""
	//DefaultClusterNodeTimeout default cluster node timeout (ms)
	//The maximum amount of time a Redis Cluster node can be unavailable, without it being considered as failing
	DefaultClusterNodeTimeout = 2000
)

// Config contains init-container settings
type Config struct {
	Port               string
	Host               string
	ConfigFile         string
	renameCommandsPath string
	renameCommandsFile string
	ClusterNodeTimeout int
}

// NewConfig builds and returns new Config instance
func NewConfig() *Config {
	return &Config{}
}

// Init used to initialize the Config
func (c *Config) Init() {
	pflag.StringVar(&c.Port, "port", RedisConfigPortDefault, "redis listen port")
	pflag.StringVar(&c.Host, "host", RedisConfigPortDefault, "redis listen host")
	pflag.StringVar(&c.ConfigFile, "c", RedisConfigFileDefault, "redis config file path")
	pflag.StringVar(&c.renameCommandsPath, "rename-command-path", RedisRenameCommandsDefaultPath, "Path to the folder where rename-commands option for redis are available")
	pflag.StringVar(&c.renameCommandsFile, "rename-command-file", RedisRenameCommandsDefaultFile, "Name of the file where rename-commands option for redis are available, disabled if empty")
	pflag.IntVar(&c.ClusterNodeTimeout, "cluster-node-timeout", DefaultClusterNodeTimeout, "redis node timeout (ms)")
}

// getRenameCommandsFile return the path to the rename command file, or empty string if not define
func (c *Config) getRenameCommandsFile() string {
	if c.renameCommandsFile == "" {
		return ""
	}
	return path.Join(c.renameCommandsPath, c.renameCommandsFile)
}
