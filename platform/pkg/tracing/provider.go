package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
)

// InitTracer initializes the global TracerProvider and returns the shutdown function
// (flush + shutdown). Call it when the application stops.
//
// Example:
//
//	shutdown, err := tracing.InitTracer(ctx, cfg)
//	if err != nil { ... }
//	defer shutdown(ctx)
func InitTracer(ctx context.Context, cfg Config) (func(context.Context) error, error) {
	samplingRatio := cfg.SamplingRatio
	if samplingRatio == 0 {
		samplingRatio = defaultSamplingRatio
	}

	// OTLP gRPC exporter — sends spans to a collector (Jaeger, Tempo, OTel Collector, ...).
	exporter, err := otlptracegrpc.New(
		ctx,
		// OTLP collector address (for example, "localhost:4317")
		otlptracegrpc.WithEndpoint(cfg.CollectorEndpoint),
		// TLS disabled — fine for local development and sidecar collectors;
		// use WithTLSCredentials in production
		otlptracegrpc.WithInsecure(),
		// Timeout of the connection attempt to the collector
		otlptracegrpc.WithTimeout(defaultTimeout),
		// gzip compression reduces the traffic to the collector
		otlptracegrpc.WithCompressor(defaultCompressor),
		// Retry with exponential backoff when span export fails
		otlptracegrpc.WithRetry(otlptracegrpc.RetryConfig{
			Enabled:         true,
			InitialInterval: defaultRetryInitInterval, // initial delay between attempts
			MaxInterval:     defaultRetryMaxInterval,  // ceiling of the exponential growth
			MaxElapsedTime:  defaultRetryMaxElapsed,   // total time after which retries stop
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("create OTLP exporter: %w", err)
	}

	// The resource describes the service the traces come from.
	// These attributes are added to every span and are visible in the tracing UI.
	res, err := resource.New(
		ctx,
		// Explicit service attributes — used to search and filter traces in the UI
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),               // service name (required attribute)
			semconv.ServiceVersion(cfg.ServiceVersion),         // version, to correlate with releases
			semconv.DeploymentEnvironmentName(cfg.Environment), // environment: production, staging, development
		),
		// Detectors collect infrastructure information automatically.
		// Useful when debugging: which host or container had the problem.
		resource.WithHost(),         // host name and host id
		resource.WithOS(),           // OS type and version
		resource.WithProcess(),      // PID, command line, Go runtime version
		resource.WithContainer(),    // container ID when running in Docker/K8s
		resource.WithTelemetrySDK(), // OpenTelemetry SDK version
	)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		// BatchSpanProcessor buffers spans and exports them in batches, which puts
		// less load on the network than SimpleSpanProcessor
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		// ParentBased — a child span is sampled whenever its parent is.
		// TraceIDRatioBased sets the sampled fraction: 1.0 = 100% (dev), ~0.1 = 10% (prod)
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(samplingRatio))),
	)

	otel.SetTracerProvider(tp)

	// The propagator defines HOW the trace ID travels between services.
	// When service A calls service B, the propagator:
	//   1. On A (inject) — writes the trace ID into the HTTP/gRPC request headers
	//   2. On B (extract) — reads it back and continues the same trace
	// Without it every service would start its own independent trace and the
	// end-to-end call chain would be invisible.
	// Both propagators are stateless strategies (implementations of TextMapPropagator).
	// They are empty structs because the header format is fixed by the standard —
	// there is nothing to configure, only the read/write algorithm to choose.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		// W3C TraceContext — the standard traceparent/tracestate headers,
		// supported by Jaeger, Tempo, Datadog and other tracing systems
		propagation.TraceContext{},
		// Baggage — arbitrary key-value pairs (user_id, tenant_id) that are
		// carried automatically through the whole call chain
		propagation.Baggage{},
	))

	// Return the shutdown function: it flushes the remaining spans and releases resources.
	return tp.Shutdown, nil
}
