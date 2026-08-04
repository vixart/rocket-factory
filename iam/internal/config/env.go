package config

type env struct {
	AppEnv         string `yaml:"app_env" env:"APP_ENV" env-default:"development"`
	ServiceName    string `yaml:"service_name"` // service name
	ServiceVersion string `yaml:"service_version" env-default:"1.0.0"`
}
