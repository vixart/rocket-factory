package tracing

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

// TraceIDFromContext extracts the trace ID of the current span in the context.
// Returns an empty string when there is no active trace (for example, when the
// TracerProvider was not initialized via InitTracer — the usual case in API tests).
func TraceIDFromContext(ctx context.Context) string {
	sc := trace.SpanFromContext(ctx).SpanContext()
	if !sc.IsValid() {
		return ""
	}

	return sc.TraceID().String()
}
