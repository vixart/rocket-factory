package config

import "fmt"

type pgConfig struct {
	Host     string `yaml:"host"     env:"POSTGRES_HOST"     env-default:"localhost"`
	Port     string `yaml:"port"     env:"POSTGRES_PORT"     env-default:"55434"`
	Database string `yaml:"database" env:"POSTGRES_DB"       env-default:"postgres"`
	User     string `yaml:"user"     env:"POSTGRES_USER"     env-default:"admin"`
	Password string `yaml:"password" env:"POSTGRES_PASSWORD" env-default:"secret"`
	SSLMode  string `yaml:"sslmode"  env:"POSTGRES_SSLMODE"  env-default:"disable"`
}

func (c *pgConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		c.Host, c.Port, c.Database, c.User, c.Password, c.SSLMode,
	)
}
