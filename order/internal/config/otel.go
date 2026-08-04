package config

// otelConfig is internal to the config package (hence the lowercase name);
// only the OTel() method of the OTelConfig interface is exposed.
type otelConfig struct {
	Endpoint   string `yaml:"endpoint"` // OTel Collector address (OTLP gRPC)
	EnableOTLP bool   `yaml:"enable_otlp" env:"ENABLE_OTLP" env-default:"true"`
}
