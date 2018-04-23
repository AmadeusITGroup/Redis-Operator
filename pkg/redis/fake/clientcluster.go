package fake

import (
	"fmt"

	radix "github.com/mediocregopher/radix.v2/redis"
)

// ClientCluster struct used to simulate in unittest the clientCluster from radix
type ClientCluster struct {
	Resps   map[string]*radix.Resp
	Resp    *radix.Resp
	Clients map[string]*radix.Client
	Client  *radix.Client
	Addrs   map[string]string
	Addr    string
	Error   error
	Servers map[string]*RedisServer
	Server  *RedisServer
}

// NewClientCluster create a new ClientCluster structure and initialize all field to nil or empty map/strings
func NewClientCluster() *ClientCluster {
	return &ClientCluster{
		Resps:   make(map[string]*radix.Resp),
		Clients: make(map[string]*radix.Client),
		Addrs:   make(map[string]string),
		Servers: make(map[string]*RedisServer),
		Addr:    "",
	}
}

// Put putss the connection back in its pool. To be used alongside any of the
// Get* methods once use of the redis.Client is done
func (c *ClientCluster) Put(conn *radix.Client) {
}

// Reset will re-retrieve the cluster topology and set up/teardown connections
// as necessary.
func (c *ClientCluster) Reset() error {
	return c.Error
}

// Cmd performs the given command on the correct cluster node and gives back the
// command's reply. The command *must* have a key parameter (i.e. len(args) >=
// 1). If any MOVED or ASK errors are returned they will be transparently
// handled by this method.
func (c *ClientCluster) Cmd(cmd string, args ...interface{}) *radix.Resp {
	command := cmd
	for _, arg := range args {
		command = fmt.Sprintf("%s %s", command, arg)
	}
	if resp, ok := c.Resps[command]; ok {
		return resp
	}

	return c.Resp
}

// GetForKey returns the Client which *ought* to handle the given key, based
// on Cluster's understanding of the cluster topology at the given moment. If
// the slot isn't known or there is an error contacting the correct node, a
// random client is returned. The client must be returned back to its pool using
// Put when through
func (c *ClientCluster) GetForKey(key string) (*radix.Client, error) {
	if client, ok := c.Clients[key]; ok {
		return client, c.Error
	}
	return c.Client, c.Error
}

// GetEvery returns a single *radix.Client per master that the cluster currently
// knows about. The map returned maps the address of the client to the client
// itself. If there is an error retrieving any of the clients (for instance if a
// new connection has to be made to get it) only that error is returned. Each
// client must be returned back to its pools using Put when through
func (c *ClientCluster) GetEvery() (map[string]*radix.Client, error) {
	return c.Clients, c.Error
}

// GetAddrForKey returns the address which would be used to handle the given key
// in the cluster.
func (c *ClientCluster) GetAddrForKey(key string) string {
	if addr, ok := c.Addrs[key]; ok {
		return addr
	}
	return c.Addr
}

// Close calls Close on all connected clients. Once this is called no other
// methods should be called on this instance of Cluster
func (c *ClientCluster) Close() {
	for _, s := range c.Servers {
		s.Close()
	}

	if c.Server != nil {
		c.Server.Close()
	}
}
