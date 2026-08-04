// Package metrics initializes the OTel MeterProvider with an OTLP gRPC exporter.
//
// It provides the platform-level metrics setup: creating the MeterProvider, connecting
// to the OTel Collector and registering the global provider.
// Business metrics (Counter, Histogram, ...) are created by each service itself.
//
// OTel metrics hierarchy:
//
//	MeterProvider — factory of Meters, owns the export settings (where and how often to send)
//	  └── Meter — a named toolbox, usually one per service or library
//	        └── Instrument — a concrete metric (Counter, UpDownCounter, Histogram, ...)
//
// By analogy: the MeterProvider is the power plant, the Meter is the breaker panel,
// and an Instrument is an individual meter on the wall.
package metrics

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
)

const (
	defaultInterval = 10 * time.Second
	shutdownTimeout = 5 * time.Second
)

var (
	provider *sdkmetric.MeterProvider
	initOnce sync.Once
)

// Init creates the OTel MeterProvider with an OTLP gRPC exporter and registers it globally.
// Metrics are pushed to the OpenTelemetry Collector.
//
// serviceName is attached to every metric through the resource.
// Functional options tune the behaviour (interval, buckets, views).
//
// After Init every Meter obtained via otel.Meter() uses this provider, including the
// platform libraries (platform/redis, for example).
func Init(cfg Config, opts ...Option) {
	initOnce.Do(func() {
		ctx := context.Background()

		o := &options{
			interval: defaultInterval,
		}
		for _, opt := range opts {
			opt(o)
		}

		// OTLP gRPC exporter — pushes metrics to the OTel Collector.
		//
		// The SDK reads these environment variables automatically:
		//   OTEL_EXPORTER_OTLP_ENDPOINT (default: https://localhost:4317)
		//   OTEL_EXPORTER_OTLP_INSECURE ("true" disables TLS)
		//   OTEL_EXPORTER_OTLP_METRICS_ENDPOINT (takes precedence over the generic one)
		//
		// WithInsecure disables TLS — for local development where the collector is
		// reachable at http://localhost:4317 without certificates.
		exporter, err := otlpmetricgrpc.New(
			ctx,
			otlpmetricgrpc.WithEndpoint(cfg.CollectorEndpoint),
			otlpmetricgrpc.WithInsecure(),
		)
		if err != nil {
			slog.Error("metrics: failed to create OTLP exporter", "error", err)
			return
		}

		// Resource — service metadata attached to every metric.
		res, err := resource.New(
			ctx,
			resource.WithAttributes(
				semconv.ServiceName(cfg.ServiceName),
				semconv.DeploymentEnvironmentName(cfg.Environment),
				semconv.ServiceInstanceID(cfg.InstanceID),
			),
		)
		if err != nil {
			slog.Error("metrics: failed to create resource", "error", err)
			return
		}

		// MeterProvider is the root object that owns collection and export.
		// PeriodicReader collects every instrument every N seconds and pushes to the exporter.
		providerOpts := []sdkmetric.Option{
			sdkmetric.WithResource(res),
			sdkmetric.WithReader(sdkmetric.NewPeriodicReader(
				exporter,
				sdkmetric.WithInterval(o.interval),
			)),
		}

		// Add the user-supplied Views, if any
		for _, view := range o.views {
			providerOpts = append(providerOpts, sdkmetric.WithView(view))
		}

		provider = sdkmetric.NewMeterProvider(providerOpts...)

		// Register the provider globally so that otel.Meter() picks it up
		otel.SetMeterProvider(provider)
	})
}

// Flush forces an export of the accumulated metrics (useful in tests before assertions).
func Flush() error {
	if provider == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	return provider.ForceFlush(ctx)
}

// Close shuts the MeterProvider down, exporting the accumulated metrics.
func Close() error {
	if provider == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	return provider.Shutdown(ctx)
}
