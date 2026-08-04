package order

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"

	"github.com/vixart/rocket-factory/order/internal/model"
	"github.com/vixart/rocket-factory/order/internal/repository/converter"
)

func (r *repository) Create(ctx context.Context, order model.Order) error {
	return r.txManager.Do(ctx, func(ctx context.Context) error {
		if err := r.createOrder(ctx, order); err != nil {
			return err
		}
		return r.createOrderItems(ctx, order)
	})
}

// createOrder inserts a row into the orders table.
func (r *repository) createOrder(ctx context.Context, order model.Order) error {
	const query = `INSERT INTO orders (uuid, user_uuid, status, created_at) VALUES ($1, $2, $3, $4)`

	_, err := r.getter.DefaultTrOrDB(ctx, r.pool).Exec(
		ctx,
		query,
		order.UUID,
		order.UserUUID,
		order.Status,
		order.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create order: %w", err)
	}

	return nil
}

// createOrderItems batch-inserts the order rows into order_items.
// order_uuid comes from order.UUID (the aggregate root); PartUUID/PartType/Price come
// from order.Items, because model.OrderItem itself has no reference to its parent.
func (r *repository) createOrderItems(ctx context.Context, order model.Order) error {
	if len(order.Items) == 0 {
		return nil
	}

	items := converter.OrderItemModelsToRecords(order.UUID, order.Items)

	query := squirrel.Insert("order_items").
		Columns("order_uuid", "part_uuid", "part_type", "price").
		PlaceholderFormat(squirrel.Dollar)

	for _, item := range items {
		query = query.Values(item.OrderUUID, item.PartUUID, item.PartType, item.Price)
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	_, err = r.getter.DefaultTrOrDB(ctx, r.pool).Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("write order items: %w", err)
	}

	return nil
}
