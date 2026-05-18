package order

import (
	"context"
	"fmt"

	errs "github.com/vixart/rocket-factory/order/internal/errors"
	"github.com/vixart/rocket-factory/order/internal/model"
	"github.com/vixart/rocket-factory/order/internal/repository/converter"
)

func (r *repository) Update(_ context.Context, order model.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.orders[order.OrderUUID]

	if !ok {
		return fmt.Errorf("заказ с uuid: %s не найден: %w", order.OrderUUID, errs.ErrOrderNotFound)
	}

	r.orders[order.OrderUUID] = converter.OrderModelToRecord(order)

	return nil
}
