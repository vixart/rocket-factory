package config

type rateLimit struct {
	RedisAddress string `yaml:"redis_address" env:"REDIS_RATE_LIMIT_ADDRESS"`
	Rate         int    `yaml:"rate" env:"REDIS_RATE_LIMIT_RATE"`
	Burst        int    `yaml:"burst" env:"REDIS_RATE_LIMIT_BURST"`
}
