package tracing

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/vixart/rocket-factory/order/internal/config"
	"github.com/vixart/rocket-factory/order/internal/model"
	"github.com/vixart/rocket-factory/order/internal/service/input"
)

type OrderService interface {
	Get(ctx context.Context, uuid uuid.UUID) (model.Order, error)
	Create(ctx context.Context, orderParts input.OrderParts) (*model.Order, error)
	Pay(ctx context.Context, orderUuid uuid.UUID, paymentMethod model.PaymentMethod) (uuid.UUID, error)
	Cancel(ctx context.Context, uuid uuid.UUID) error
}

type tracingOrderService struct {
	OrderService
}

func NewTracingOrderService(next OrderService) OrderService {
	return &tracingOrderService{OrderService: next}
}

func (s *tracingOrderService) Create(ctx context.Context, orderParts input.OrderParts) (*model.Order, error) {
	ctx, span := otel.Tracer(config.AppConfig().Env.ServiceName).Start(ctx, "order.Create")
	defer span.End()

	span.SetAttributes(
		attribute.String("order.hull_uuid", orderParts.HullUUID.String()),
		attribute.String("order.engine_uuid", orderParts.EngineUUID.String()),
	)

	order, err := s.OrderService.Create(ctx, orderParts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(
		attribute.String("order.uuid", order.UUID.String()),
		attribute.Int("order.items_count", len(order.Items)),
		attribute.Int64("order.total_price", order.TotalPrice()),
	)

	return order, nil
}

func (s *tracingOrderService) Pay(ctx context.Context, orderUuid uuid.UUID, paymentMethod model.PaymentMethod) (uuid.UUID, error) {
	ctx, span := otel.Tracer(config.AppConfig().Env.ServiceName).Start(ctx, "order.Pay")
	defer span.End()

	span.SetAttributes(
		attribute.String("order.uuid", orderUuid.String()),
		attribute.String("order.payment_method", string(paymentMethod)),
	)

	transactionUUID, err := s.OrderService.Pay(ctx, orderUuid, paymentMethod)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return uuid.Nil, err
	}

	span.SetAttributes(
		attribute.String("order.transaction_uuid", transactionUUID.String()),
	)

	return transactionUUID, nil
}
