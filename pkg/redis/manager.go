package redis

import "github.com/amadeusitgroup/redis-operator/pkg/api/redis/v1"

// Manager regroups Function for managing a Redis Cluster
type Manager struct {
	admin *Admin
}

// NewManager builds and returns new Manager instance
func NewManager(admin *Admin) *Manager {
	return &Manager{
		admin: admin,
	}
}

// BuildClusterStatus builds and returns new instance of the RedisClusterClusterStatus
func (m *Manager) BuildClusterStatus() (*v1.RedisClusterClusterStatus, error) {
	status := &v1.RedisClusterClusterStatus{}

	return status, nil
}
