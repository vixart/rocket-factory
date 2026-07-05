package config

type env struct {
	AppEnv         string `yaml:"app_env" env:"APP_ENV" env-default:"development"`
	ServiceName    string `yaml:"service_name"` // имя сервиса
	ServiceVersion string `yaml:"service_version" env-default:"1.0.0"`
}
