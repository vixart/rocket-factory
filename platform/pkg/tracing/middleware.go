package tracing

import (
	"net/http"

	"go.opentelemetry.io/otel/trace"
)

func TraceIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := trace.SpanFromContext(r.Context())
		traceID := span.SpanContext().TraceID().String()
		if traceID != "" {
			w.Header().Set("X-Trace-ID", traceID)
		}
		next.ServeHTTP(w, r)
	})
}
