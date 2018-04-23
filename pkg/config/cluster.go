package config

import "github.com/spf13/pflag"

// Cluster used to store all Redis Cluster configuration information
type Cluster struct {
	Namespace   string
	NodeService string
}

// AddFlags use to add the Redis-Cluster Config flags to the command line
func (c *Cluster) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.Namespace, "ns", "", "redis-node k8s namespace")
	fs.StringVar(&c.NodeService, "rs", "", "redis-node k8s service name")

}
