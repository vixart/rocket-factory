package config

import "net"

type httpConfig struct {
	Host string `yaml:"host" env:"HTTP_HOST" env-default:"localhost"`
	Port string `yaml:"port" env:"HTTP_PORT" env-default:"8080"`
}

func (c *httpConfig) Address() string {
	return net.JoinHostPort(c.Host, c.Port)
}
