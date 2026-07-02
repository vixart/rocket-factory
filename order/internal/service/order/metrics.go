package order

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var meter = otel.Meter("order-service")

var (
	ordersCreatedTotal, _ = meter.Int64Counter("orders_created",
		metric.WithDescription("Количество созданных заказов"),
	)

	ordersPaidTotal, _ = meter.Int64Counter("orders_paid",
		metric.WithDescription("Количество оплаченных заказов"),
	)

	ordersRevenueTotal, _ = meter.Int64Counter("orders_revenue",
		metric.WithDescription("Суммарная выручка (в копейках)"),
	)
)
