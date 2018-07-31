package redisnode

import (
	"time"

	"github.com/amadeusitgroup/redis-operator/pkg/config"
	"github.com/spf13/pflag"
)

const (
	// RedisStartWaitDefault default stop duration (sec)
	RedisStartWaitDefault = 10 * time.Second
	// RedisStartDelayDefault default start delay duration (sec)
	RedisStartDelayDefault = 10 * time.Second
	// RedisServerBinDefault default binary name
	RedisServerBinDefault = "redis-server"
	// RedisServerPortDefault default redis port
	RedisServerPortDefault = "6379"
	// HTTPServerAddrDefault default http server address
	HTTPServerAddrDefault = "0.0.0.0:8080"
	// RedisMaxMemoryDefault default redis max memory
	RedisMaxMemoryDefault = 0
	// RedisMaxMemoryPolicyDefault default redis max memory evition policy
	RedisMaxMemoryPolicyDefault = "noeviction"
)

// Config contains configuration for redis-operator
type Config struct {
	KubeConfigFile       string
	Master               string
	Redis                config.Redis
	Cluster              config.Cluster
	RedisServerBin       string
	RedisServerPort      string
	RedisStartWait       time.Duration
	RedisStartDelay      time.Duration
	HTTPServerAddr       string
	RedisMaxMemory       uint32
	RedisMaxMemoryPolicy string
}

// NewRedisNodeConfig builds and returns a redis-operator Config
func NewRedisNodeConfig() *Config {

	return &Config{}
}

// AddFlags add cobra flags to populate Config
func (c *Config) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.KubeConfigFile, "kubeconfig", c.KubeConfigFile, "Location of kubecfg file for access to kubernetes master service")
	fs.StringVar(&c.Master, "master", c.Master, "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	fs.DurationVar(&c.RedisStartWait, "t", RedisStartWaitDefault, "Max time waiting for redis to start")
	fs.DurationVar(&c.RedisStartDelay, "d", RedisStartDelayDefault, "delay before that the redis-server is started")

	fs.StringVar(&c.RedisServerBin, "bin", RedisServerBinDefault, "redis server binary file name")
	fs.StringVar(&c.RedisServerPort, "port", RedisServerPortDefault, "redis server listen port")
	fs.Uint32Var(&c.RedisMaxMemory, "max-memory", RedisMaxMemoryDefault, "redis max memory")
	fs.StringVar(&c.RedisMaxMemoryPolicy, "max-memory-policy", RedisMaxMemoryPolicyDefault, "redis max memory evition policy")

	fs.StringVar(&c.HTTPServerAddr, "http-addr", HTTPServerAddrDefault, "the http server listen address")

	c.Redis.AddFlags(fs)
	c.Cluster.AddFlags(fs)
}
