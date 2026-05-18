package order

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	errs "github.com/vixart/rocket-factory/order/internal/errors"
	"github.com/vixart/rocket-factory/order/internal/model"
	"github.com/vixart/rocket-factory/order/internal/repository/converter"
)

func (r *repository) Get(_ context.Context, uuid uuid.UUID) (model.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	order, ok := r.orders[uuid]
	if !ok {
		return model.Order{}, fmt.Errorf("заказ с uuid: %s не найден: %w", uuid, errs.ErrOrderNotFound)
	}

	return converter.OrderRecordToModel(order), nil
}
