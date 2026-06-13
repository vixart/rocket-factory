package order

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	errs "github.com/vixart/rocket-factory/order/internal/errors"
	"github.com/vixart/rocket-factory/order/internal/model"
)

func (r *repository) GetForUpdate(ctx context.Context, uuid uuid.UUID) (model.Order, error) {
	order, err := r.getOrderForUpdate(ctx, uuid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Order{}, fmt.Errorf("заказ не найден: %w", errs.ErrOrderNotFound)
		}
		return model.Order{}, err
	}

	orderItems, err := r.getOrderItems(ctx, uuid)
	if err != nil {
		return model.Order{}, err
	}

	order.Items = orderItems

	return order, nil
}
