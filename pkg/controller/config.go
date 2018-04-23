package controller

import "github.com/amadeusitgroup/redis-operator/pkg/config"

// Config contains the Controller settings
type Config struct {
	NbWorker int
	redis    config.Redis
}

// NewConfig builds and returns new Config instance
func NewConfig(nbWorker int, redis config.Redis) *Config {
	return &Config{
		NbWorker: nbWorker,
		redis:    redis,
	}
}
