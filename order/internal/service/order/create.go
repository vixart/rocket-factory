package order

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	errs "github.com/vixart/rocket-factory/order/internal/errors"
	"github.com/vixart/rocket-factory/order/internal/model"
)

func (s *service) Create(ctx context.Context, orderParts model.OrderParts) (*model.Order, error) {
	partsUuids := []uuid.UUID{orderParts.HullUUID, orderParts.EngineUUID}

	if orderParts.ShieldUUID != nil {
		partsUuids = append(partsUuids, *orderParts.ShieldUUID)
	}

	if orderParts.WeaponUUID != nil {
		partsUuids = append(partsUuids, *orderParts.WeaponUUID)
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	parts, err := s.inventoryClient.ListParts(ctxWithTimeout, partsUuids)
	if err != nil {
		return nil, fmt.Errorf("при создании заказа не удалось получить детали: %w", err)
	}

	if len(parts) != len(partsUuids) {
		return nil, errs.ErrPartNotFound
	}

	totalPrice := int64(0)

	for _, p := range parts {
		if p.StockQuantity <= 0 {
			return nil, fmt.Errorf("детали нет на складе: %s | %w", p.UUID, errs.ErrPartInsufficientStock)
		}
		totalPrice = totalPrice + p.Price
	}

	orderUUID := uuid.New()
	order := model.Order{
		OrderUUID:  orderUUID,
		HullUUID:   orderParts.HullUUID,
		EngineUUID: orderParts.EngineUUID,
		ShieldUUID: orderParts.ShieldUUID,
		WeaponUUID: orderParts.WeaponUUID,
		TotalPrice: totalPrice,
		Status:     model.OrderStatusPendingPayment,
		CreatedAt:  time.Now(),
	}

	err = s.orderRepository.Create(ctx, order)
	if err != nil {
		return nil, err
	}

	return &order, nil
}
