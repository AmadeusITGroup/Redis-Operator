package admin

import (
	"github.com/amadeusitgroup/redis-operator/pkg/redis"

	radix "github.com/mediocregopher/radix.v2/redis"
)

// Connections fake redis connection handler, do nothing
type Connections struct {
	clients map[string]redis.ClientInterface
}

// Close used to close all possible resources instanciate by the Connections
func (cnx *Connections) Close() {
}

// Add connect to the given address and
// register the client connection to the map
func (cnx *Connections) Add(addr string) error {
	return nil
}

// Remove disconnect and remove the client connection from the map
func (cnx *Connections) Remove(addr string) {
}

// Get returns a client connection for the given adress,
// connects if the connection is not in the map yet
func (cnx *Connections) Get(addr string) (redis.ClientInterface, error) {
	return nil, nil
}

// GetRandom returns a client connection to a random node of the client map
func (cnx *Connections) GetRandom() (redis.ClientInterface, error) {
	return nil, nil
}

// GetDifferentFrom returns random a client connection different from given address
func (cnx *Connections) GetDifferentFrom(addr string) (redis.ClientInterface, error) {
	return nil, nil
}

// GetAll returns a map of all clients per address
func (cnx *Connections) GetAll() map[string]redis.ClientInterface {
	return cnx.clients
}

// GetSelected returns a map of all clients per address
func (cnx *Connections) GetSelected(addrs []string) map[string]redis.ClientInterface {
	return cnx.clients
}

// Reconnect force a reconnection on the given address
// is the adress is not part of the map, act like Add
func (cnx *Connections) Reconnect(addr string) error {
	return nil
}

// AddAll connect to the given list of addresses and
// register them in the map
// fail silently
func (cnx *Connections) AddAll(addrs []string) {
}

// ReplaceAll clear the pool and re-populate it with new connections
// fail silently
func (cnx *Connections) ReplaceAll(addrs []string) {
}

// Reset close all connections and clear the connection map
func (cnx *Connections) Reset() {
}

// ValidateResp check the redis resp, eventually reconnect on connection error
// in case of error, customize the error, log it and return it
func (cnx *Connections) ValidateResp(resp *radix.Resp, addr, errMessage string) error {
	return nil
}

// ValidatePipeResp wait for all answers in the pipe and validate the response
// in case of network issue clear the pipe and return
// in case of error, return false
func (cnx *Connections) ValidatePipeResp(client redis.ClientInterface, addr, errMessage string) bool {
	return true
}
