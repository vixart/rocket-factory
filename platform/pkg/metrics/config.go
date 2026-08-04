package metrics

// Config holds the metrics initialization settings.
type Config struct {
	// ServiceName is attached to every metric through the resource (service.name).
	ServiceName string
	// Environment is the deployment environment ("production", "staging", "development").
	Environment string
	// InstanceID is the host name.
	InstanceID string
	// CollectorEndpoint is the OTLP collector address (for example, "localhost:4317").
	// Falls back to defaultOTLPEndpoint when empty.
	CollectorEndpoint string
}
