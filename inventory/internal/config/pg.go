package config

import "fmt"

type pgConfig struct {
	Host     string `yaml:"host"     env:"POSTGRES_HOST"     env-default:"localhost"`
	Port     string `yaml:"port"     env:"POSTGRES_PORT"     env-default:"5432"`
	Database string `yaml:"database" env:"POSTGRES_DB"       env-default:"postgres"`
	User     string `yaml:"user"     env:"POSTGRES_USER"     env-default:"admin"`
	Password string `yaml:"password" env:"POSTGRES_PASSWORD" env-default:"secret"`
	SSLMode  string `yaml:"sslmode"  env:"POSTGRES_SSLMODE"  env-default:"disable"`

	// MaxConns is the pgx pool size. Without it pgxpool uses its own default of
	// max(4, NumCPU), i.e. 4–8 connections for the whole service. Reserve holds a
	// connection while waiting for a row lock in SELECT ... FOR UPDATE, so on such a
	// pool a few concurrent reservations drain it completely and even fast reads queue
	// up waiting for a free connection.
	MaxConns int `yaml:"max_conns" env:"POSTGRES_MAX_CONNS" env-default:"25"`
}

func (c *pgConfig) DSN() string {
	dsn := fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		c.Host, c.Port, c.Database, c.User, c.Password, c.SSLMode,
	)

	// pgx treats pool_max_conns=0 as a configuration error, so a non-positive value
	// leaves the parameter out and the pool keeps the pgx default.
	if c.MaxConns > 0 {
		dsn += fmt.Sprintf(" pool_max_conns=%d", c.MaxConns)
	}

	return dsn
}
