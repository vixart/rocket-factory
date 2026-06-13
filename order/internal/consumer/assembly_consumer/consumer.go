package assembly_consumer

import (
	"context"
	"log/slog"

	"github.com/vixart/rocket-factory/order/internal/service/order"
)

type service struct {
	shipAssembledConsumer Consumer
	orderRepository       order.Repository
	inventoryClient       order.InventoryClient
}

func NewService(consumer Consumer, repository order.Repository, client order.InventoryClient) *service {
	return &service{
		shipAssembledConsumer: consumer,
		orderRepository:       repository,
		inventoryClient:       client,
	}
}

func (s *service) RunConsumer(ctx context.Context) error {
	slog.InfoContext(ctx, "запуск потребителя ShipAssembled")

	return s.shipAssembledConsumer.Consume(ctx, s.ShipAssembledHandler)
}
