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

	for _, p := range parts {
		if p.StockQuantity <= 0 {
			return nil, fmt.Errorf("детали нет на складе: %s | %w", p.UUID, errs.ErrPartInsufficientStock)
		}
	}

	// обязательные части
	enginePart := parts[orderParts.EngineUUID]
	hullPart := parts[orderParts.HullUUID]

	// опциональные
	var shieldPart, weaponPart *model.Part

	if orderParts.ShieldUUID != nil {
		part := parts[*orderParts.ShieldUUID]
		shieldPart = &part
	}

	if orderParts.WeaponUUID != nil {
		part := parts[*orderParts.WeaponUUID]
		weaponPart = &part
	}

	totalPrice := enginePart.Price + hullPart.Price
	if shieldPart != nil {
		totalPrice += shieldPart.Price
	}
	if weaponPart != nil {
		totalPrice += weaponPart.Price
	}

	orderUUID := uuid.New()
	order := model.Order{
		OrderUUID:  orderUUID,
		HullUUID:   hullPart.UUID,
		EngineUUID: enginePart.UUID,
		TotalPrice: totalPrice,
		Status:     model.OrderStatusPendingPayment,
		CreatedAt:  time.Now(),
	}
	if shieldPart != nil {
		order.ShieldUUID = new(shieldPart.UUID)
	}
	if weaponPart != nil {
		order.WeaponUUID = new(weaponPart.UUID)
	}

	err = s.orderRepository.Create(ctx, order)
	if err != nil {
		return nil, err
	}

	return &order, nil
}
