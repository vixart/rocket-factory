package tracing

import "time"

const (
	// defaultCompressor is the compression algorithm of the OTLP exporter.
	defaultCompressor = "gzip"

	// defaultTimeout bounds the connection attempt to the collector.
	defaultTimeout = 5 * time.Second

	// Retry policy of the exporter when sending spans fails.
	defaultRetryInitInterval = 500 * time.Millisecond // initial interval between attempts
	defaultRetryMaxInterval  = 5 * time.Second        // maximum interval (exponential backoff)
	defaultRetryMaxElapsed   = 30 * time.Second       // maximum total retry time

	// defaultSamplingRatio is the fraction of sampled traces (1.0 = 100%).
	defaultSamplingRatio = 1.0

	// TraceIDHeader carries the trace ID back to the client in the gRPC response
	// so that the trace can be looked up in Jaeger/Tempo.
	TraceIDHeader = "x-trace-id"
)

// Config holds the tracer initialization settings.
type Config struct {
	// CollectorEndpoint is the OTLP collector address (for example, "localhost:4317").
	CollectorEndpoint string
	// ServiceName is the service name shown in traces.
	ServiceName string
	// Environment is the deployment environment ("production", "staging").
	Environment string
	// ServiceVersion is the service version (for example, "1.0.0").
	ServiceVersion string
	// SamplingRatio is the sampled fraction of traces, 0.0–1.0:
	// 1.0 = every trace (development), 0.1 = 10% (production).
	// When left at 0, defaultSamplingRatio (100%) is used.
	SamplingRatio float64
}
