package order

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	errs "github.com/vixart/rocket-factory/order/internal/errors"
	"github.com/vixart/rocket-factory/order/internal/model"
	"github.com/vixart/rocket-factory/order/internal/service/input"
	"github.com/vixart/rocket-factory/platform/pkg/auth"
)

func (s *service) Create(ctx context.Context, orderParts input.OrderParts) (*model.Order, error) {
	userUUID, ok := auth.UserUUIDFromContext(ctx)
	if !ok {
		return nil, errs.ErrUnauthorized
	}

	partsUuids := []uuid.UUID{orderParts.HullUUID, orderParts.EngineUUID}

	if orderParts.ShieldUUID != nil {
		partsUuids = append(partsUuids, *orderParts.ShieldUUID)
	}

	if orderParts.WeaponUUID != nil {
		partsUuids = append(partsUuids, *orderParts.WeaponUUID)
	}

	err := validateUniqueUUIDs(partsUuids)
	if err != nil {
		slog.WarnContext(ctx, "заказ отклонён: детали повторяются",
			"user_uuid", userUUID, "parts_count", len(partsUuids), "error", err)
		return nil, err
	}

	slog.DebugContext(ctx, "создание заказа: запрошены детали",
		"user_uuid", userUUID, "part_uuids", partsUuids)

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	parts, err := s.inventoryClient.ListParts(ctxWithTimeout, partsUuids)
	if err != nil {
		return nil, fmt.Errorf("при создании заказа не удалось получить детали: %w", err)
	}

	if len(parts) != len(partsUuids) {
		slog.WarnContext(ctx, "заказ отклонён: деталь не найдена",
			"user_uuid", userUUID, "requested", len(partsUuids), "found", len(parts))
		return nil, errs.ErrPartNotFound
	}

	orderItems := make([]model.OrderItem, 0, len(parts))
	for _, p := range parts {
		if p.StockQuantity <= 0 {
			slog.WarnContext(ctx, "заказ отклонён: детали нет на складе",
				"user_uuid", userUUID, "part_uuid", p.UUID, "part_type", p.PartType)
			return nil, fmt.Errorf("детали нет на складе: %s | %w", p.UUID, errs.ErrOutOfStock)
		}
		orderItems = append(orderItems, model.OrderItem{
			UUID:     p.UUID,
			PartType: p.PartType,
			Price:    p.Price,
		})
	}

	err = s.inventoryClient.ValidateCompatibility(ctx, orderParts)
	if err != nil {
		slog.WarnContext(ctx, "заказ отклонён: детали несовместимы",
			"user_uuid", userUUID, "part_uuids", partsUuids, "error", err)
		return nil, err
	}

	err = s.inventoryClient.ReserveParts(ctx, partsUuids)
	if err != nil {
		return nil, err
	}

	slog.DebugContext(ctx, "детали зарезервированы", "user_uuid", userUUID, "part_uuids", partsUuids)

	orderUUID := uuid.New()
	order := model.Order{
		UUID:      orderUUID,
		UserUUID:  userUUID,
		Items:     orderItems,
		Status:    model.OrderStatusPendingPayment,
		CreatedAt: time.Now(),
	}

	err = s.orderRepository.Create(ctx, order)
	if err != nil {
		return nil, err
	}

	ordersCreatedTotal.Add(ctx, 1)

	slog.InfoContext(ctx, "заказ создан",
		"order_uuid", orderUUID,
		"user_uuid", userUUID,
		"items_count", len(orderItems),
		"total_price", order.TotalPrice(),
	)

	return &order, nil
}

func validateUniqueUUIDs(ids []uuid.UUID) error {
	seen := make(map[uuid.UUID]struct{}, len(ids))

	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}

		if _, ok := seen[id]; ok {
			return fmt.Errorf(
				"uuid %s задублирован в наборе деталей: %w",
				id,
				errs.ErrInvalidUUID,
			)
		}

		seen[id] = struct{}{}
	}

	return nil
}
