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
	txManager             TxManager
}

func NewService(
	consumer Consumer,
	repository order.Repository,
	client order.InventoryClient,
	txManager TxManager,
) *service {
	return &service{
		shipAssembledConsumer: consumer,
		orderRepository:       repository,
		inventoryClient:       client,
		txManager:             txManager,
	}
}

func (s *service) RunConsumer(ctx context.Context) error {
	slog.InfoContext(ctx, "starting the ShipAssembled consumer")

	return s.shipAssembledConsumer.Consume(ctx, s.ShipAssembledHandler)
}
