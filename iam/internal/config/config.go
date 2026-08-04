package config

import (
	"flag"
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

var appConfig *config

type config struct {
	Env     env           `yaml:"env"`
	GRPC    grpcConfig    `yaml:"grpc"`
	Logger  loggerConfig  `yaml:"logger"`
	PG      pgConfig      `yaml:"pg"`
	Redis   redisConfig   `yaml:"redis"`
	Session sessionConfig `yaml:"session"`
	Otel    otelConfig    `yaml:"otel"`
}

func MustLoad() {
	configPath := ResolveConfigPath()

	cfg, err := Load(configPath)
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	appConfig = cfg
}

func AppConfig() *config {
	return appConfig
}

const defaultConfigPath = "config.local.yaml"

// ResolveConfigPath resolves the config file path by priority:
// -config flag > CONFIG_PATH env var > "config.local.yaml".
func ResolveConfigPath() string {
	var cfgFlag string
	flag.StringVar(&cfgFlag, "config", "", "path to the YAML config (for example, config.staging.yaml)")
	flag.Parse()

	if cfgFlag != "" {
		return cfgFlag
	}

	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		return envPath
	}

	return defaultConfigPath
}

func Load(path string) (*config, error) {
	var cfg config

	if path != "" {
		// ReadConfig reads the YAML file and then overrides values from environment variables.
		// Priority: env > yaml > env-default
		if err := cleanenv.ReadConfig(path, &cfg); err != nil {
			return nil, fmt.Errorf("failed to load config from %q: %w", path, err)
		}

		return &cfg, nil
	}

	// With no path given, read from environment variables only
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("failed to load config from env: %w", err)
	}

	return &cfg, nil
}
