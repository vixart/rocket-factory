package config

// otelConfig — внутренняя структура пакета config (имя с маленькой буквы),
// наружу торчит только метод OTel() OTelConfig интерфейса.
type otelConfig struct {
	Endpoint   string `yaml:"endpoint"` // адрес OTEL Collector (OTLP gRPC)
	EnableOTLP bool   `yaml:"enable_otlp" env:"ENABLE_OTLP" env-default:"true"`
}
