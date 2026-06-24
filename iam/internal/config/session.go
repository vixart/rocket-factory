package config

import "time"

type sessionConfig struct {
	TTL time.Duration `yaml:"ttl" env:"SESSION_TTL" env-default:"24h"`
}
