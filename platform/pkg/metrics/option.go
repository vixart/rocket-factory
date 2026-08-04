package metrics

import (
	"time"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// options holds the MeterProvider settings.
type options struct {
	// interval is how often metrics are exported to the collector.
	// It defaults to 10 seconds instead of the OTel SDK default of 60, so that
	// metrics show up in Prometheus/Grafana quickly during local development.
	interval time.Duration

	// views override the default aggregation of specific instruments,
	// for example custom histogram buckets.
	views []sdkmetric.View
}

// Option is a functional option for the MeterProvider.
type Option func(*options)

// WithInterval sets the metric export interval.
// The default of 10 seconds suits local development;
// 15–60 seconds is recommended in production.
func WithInterval(d time.Duration) Option {
	return func(o *options) {
		if d > 0 {
			o.interval = d
		}
	}
}

// WithView adds a View that overrides the aggregation of a specific metric.
//
// Example — custom buckets for the rpc.server.call.duration histogram:
//
//	metrics.WithView(sdkmetric.NewView(
//	    sdkmetric.Instrument{Name: "rpc.server.call.duration"},
//	    sdkmetric.Stream{
//	        Aggregation: sdkmetric.AggregationExplicitBucketHistogram{
//	            Boundaries: []float64{0.0001, 0.001, 0.01, 0.1, 0.5, 1, 5},
//	        },
//	    },
//	))
func WithView(v sdkmetric.View) Option {
	return func(o *options) {
		o.views = append(o.views, v)
	}
}
