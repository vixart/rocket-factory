package logger

import "time"

const (
	// defaultOTLPEndpoint is the default gRPC address of the OTLP collector.
	defaultOTLPEndpoint = "localhost:4317"

	// defaultShutdownTimeout bounds the final log flush on application shutdown.
	defaultShutdownTimeout = 2 * time.Second
)

// Config holds the logger initialization settings.
type Config struct {
	// Level is the log level: "debug", "info", "warn", "error".
	// An invalid value falls back to "info".
	Level string
	// ServiceName is attached to every log record through the resource (service.name).
	ServiceName string
	// Environment is the deployment environment ("production", "staging", "development").
	Environment string
	// EnableOTLP also ships logs to the OTLP collector on top of stdout.
	// When false, logs go to stdout only, in JSON format.
	EnableOTLP bool
	// CollectorEndpoint is the OTLP collector address (for example, "localhost:4317").
	// Falls back to defaultOTLPEndpoint when empty.
	CollectorEndpoint string
}
