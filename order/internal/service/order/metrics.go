package order

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var meter = otel.Meter("order-service")

var (
	ordersCreatedTotal, _ = meter.Int64Counter("orders_created",
		metric.WithDescription("Number of created orders"),
	)

	ordersPaidTotal, _ = meter.Int64Counter("orders_paid",
		metric.WithDescription("Number of paid orders"),
	)

	ordersRevenueTotal, _ = meter.Int64Counter("orders_revenue",
		metric.WithDescription("Total revenue in kopecks"),
	)
)
