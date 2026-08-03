package config

import "fmt"

type pgConfig struct {
	Host     string `yaml:"host"     env:"POSTGRES_HOST"     env-default:"localhost"`
	Port     string `yaml:"port"     env:"POSTGRES_PORT"     env-default:"5432"`
	Database string `yaml:"database" env:"POSTGRES_DB"       env-default:"postgres"`
	User     string `yaml:"user"     env:"POSTGRES_USER"     env-default:"admin"`
	Password string `yaml:"password" env:"POSTGRES_PASSWORD" env-default:"secret"`
	SSLMode  string `yaml:"sslmode"  env:"POSTGRES_SSLMODE"  env-default:"disable"`

	// MaxConns — размер пула pgx. Без него pgxpool берёт свой дефолт
	// max(4, NumCPU), то есть 4–8 соединений на весь сервис. Reserve держит
	// соединение, пока ждёт блокировку строки в SELECT ... FOR UPDATE, поэтому
	// на таком пуле несколько параллельных резервирований выгребают его досуха,
	// и даже быстрые чтения встают в очередь за свободным соединением.
	MaxConns int `yaml:"max_conns" env:"POSTGRES_MAX_CONNS" env-default:"25"`
}

func (c *pgConfig) DSN() string {
	dsn := fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		c.Host, c.Port, c.Database, c.User, c.Password, c.SSLMode,
	)

	// pool_max_conns=0 pgx считает ошибкой конфигурации, поэтому при неположительном
	// значении параметр не добавляем — пул останется на дефолте pgx.
	if c.MaxConns > 0 {
		dsn += fmt.Sprintf(" pool_max_conns=%d", c.MaxConns)
	}

	return dsn
}
