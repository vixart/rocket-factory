// Package logger is the platform logging component: stdout (JSON) + OTLP → Elasticsearch.
//
// Design: slog with a fanout handler that duplicates every record into two
// destinations:
//  1. stdout — JSON format for local development and kubectl logs
//  2. OTLP gRPC → OTel Collector → Elasticsearch → Kibana — centralized storage
//
// The component follows the same pattern as tracing and metrics: Init + Close,
// configuration through Config, graceful degradation when the collector is unavailable.
package logger

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"

	slogmulti "github.com/samber/slog-multi"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	otelLogSdk "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
)

var (
	// otelProvider is kept for a clean shutdown (Close).
	otelProvider *otelLogSdk.LoggerProvider

	// initOnce guarantees Init runs exactly once (thread safety).
	initOnce sync.Once
)

// Init builds the global slog logger and installs it via slog.SetDefault.
//
// With cfg.EnableOTLP=true logs are additionally shipped to the OTLP collector
// (usually OTel Collector → Elasticsearch → Kibana).
//
// Example:
//
//	logger.Init(logger.Config{
//	    Level:             "info",
//	    ServiceName:       "ufo-service",
//	    Environment:       "development",
//	    EnableOTLP:        true,
//	    CollectorEndpoint: "localhost:4317",
//	})
//	defer logger.Close()
func Init(cfg Config) {
	initOnce.Do(func() {
		level := parseLevel(cfg.Level)

		// Primary handler — JSON output to stdout, always enabled.
		// AddSource: true adds the calling file and line to every record.
		stdoutHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:     level, // minimum log level (DEBUG, INFO, WARN, ERROR)
			AddSource: true,  // enables the "source" field with file and line
		})

		// Resulting handler: stdout only, or a fanout of stdout + OTLP.
		handler := slog.Handler(stdoutHandler)

		// When OTLP is enabled and reachable, Fanout duplicates every record into both handlers.
		if cfg.EnableOTLP {
			otelHandler := newOTLPExportHandler(cfg)
			// Graceful degradation: if OTLP is unavailable otelHandler is nil and we stay on stdout.
			if otelHandler != nil {
				// otelslog.Handler lets records of every level through, so the cfg.Level filter
				// is applied here: otherwise DEBUG records would reach Kibana at level=info
				// even though stdout does not show them.
				handler = slogmulti.Fanout(stdoutHandler, withMinLevel(otelHandler, level))
			}
		}

		slog.SetDefault(slog.New(handler))
	})
}

// Close shuts the OTLP provider down, flushing the remaining logs.
// Call it when the application stops, usually via defer.
func Close() error {
	if otelProvider == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
	defer cancel()

	return otelProvider.Shutdown(ctx)
}

// newOTLPExportHandler creates an OTLP handler with a gRPC exporter.
// It returns nil on error (graceful degradation — logs keep going to stdout).
func newOTLPExportHandler(cfg Config) slog.Handler {
	ctx := context.Background()

	endpoint := cmp.Or(cfg.CollectorEndpoint, defaultOTLPEndpoint)

	// gRPC exporter — ships logs to an OTLP collector (OTel Collector and friends).
	exporter, err := otlploggrpc.New(
		ctx,
		otlploggrpc.WithEndpoint(endpoint), // collector address (host:port)
		otlploggrpc.WithInsecure(),         // no TLS (local development)
	)
	if err != nil {
		// slog may not be initialized yet, so write to stderr directly
		fmt.Fprintf(os.Stderr, "logger: failed to create OTLP exporter: %v\n", err)
		return nil
	}

	// Resource — service metadata attached to every log record.
	// Used for filtering in Kibana: service.name, deployment.environment.
	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			attribute.String("deployment.environment", cfg.Environment),
		),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger: failed to create OTel resource: %v\n", err)
		return nil
	}

	// LoggerProvider owns the exporter lifecycle and log batching.
	//
	// BatchProcessor queues records and ships them in batches instead of one by one.
	// Defaults from the OTel SDK:
	//   - MaxExportBatchSize = 512   — maximum records per batch
	//   - ExportInterval     = 1s    — how often the queue is flushed
	//   - ExportTimeout      = 30s   — timeout of a single batch export
	//   - MaxQueueSize       = 2048  — internal queue size; records are dropped on overflow
	provider := otelLogSdk.NewLoggerProvider(
		otelLogSdk.WithResource(res),
		otelLogSdk.WithProcessor(otelLogSdk.NewBatchProcessor(exporter)),
	)
	otelProvider = provider // kept for Shutdown on application exit

	// otelslog.NewHandler is the official OpenTelemetry bridge that converts slog
	// records into OTel Log Records (severity and attribute mapping).
	return otelslog.NewHandler(
		"app",
		otelslog.WithLoggerProvider(provider),
	)
}

// minLevelHandler is a wrapper that drops records below the given level.
// Needed for handlers without their own level setting (otelslog).
type minLevelHandler struct {
	slog.Handler
	level slog.Level
}

// withMinLevel wraps a handler with a minimum level filter.
func withMinLevel(h slog.Handler, level slog.Level) slog.Handler {
	return minLevelHandler{Handler: h, level: level}
}

func (h minLevelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.level && h.Handler.Enabled(ctx, level)
}

// WithAttrs and WithGroup must return the wrapper as well, otherwise slog loses the
// filter on derived loggers (slog.With(...)).
func (h minLevelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return minLevelHandler{Handler: h.Handler.WithAttrs(attrs), level: h.level}
}

func (h minLevelHandler) WithGroup(name string) slog.Handler {
	return minLevelHandler{Handler: h.Handler.WithGroup(name), level: h.level}
}

// parseLevel converts a textual level ("debug", "info", "warn", "error") into slog.Level.
// An invalid value falls back to INFO as a safe default.
func parseLevel(s string) slog.Level {
	var level slog.Level
	if err := level.UnmarshalText([]byte(s)); err != nil {
		return slog.LevelInfo
	}

	return level
}
