package tracing

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// TraceIDUnaryServerInterceptor adds the trace ID to the gRPC response headers
// so that the client can look the trace up in Jaeger/Tempo.
//
// Graceful degradation: when the TracerProvider is not initialized (InitTracer
// was not called or tracing is disabled in the config — for example in bufconn
// API tests), TraceIDFromContext returns an empty string and no header is set.
// That is expected behaviour: the handler call proceeds normally.
func TraceIDUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		traceID := TraceIDFromContext(ctx)
		if traceID != "" {
			// grpc.SetHeader puts metadata into the gRPC response headers,
			// which the client receives together with the call result.
			if err := grpc.SetHeader(ctx, metadata.Pairs(TraceIDHeader, traceID)); err != nil {
				slog.WarnContext(ctx, "failed to set the trace ID response header", slog.String("error", err.Error()))
			}
		}

		return handler(ctx, req)
	}
}
