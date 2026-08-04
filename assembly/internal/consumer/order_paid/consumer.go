package order_paid

import (
	"context"
	"log/slog"
)

type service struct {
	orderPaidConsumer   Consumer
	shipAssembleService ShipAssembleService
}

func NewService(orderPaidConsumer Consumer, shipAssembleService ShipAssembleService) *service {
	return &service{
		orderPaidConsumer:   orderPaidConsumer,
		shipAssembleService: shipAssembleService,
	}
}

func (s *service) RunConsumer(ctx context.Context) error {
	slog.InfoContext(ctx, "starting the OrderPaid consumer")

	return s.orderPaidConsumer.Consume(ctx, s.OrderPaidHandler)
}
