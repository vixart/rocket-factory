package order

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"

	"github.com/vixart/rocket-factory/order/internal/model"
	"github.com/vixart/rocket-factory/order/internal/repository/converter"
)

func (r *repository) Create(ctx context.Context, order model.Order) error {
	return r.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := r.createOrder(txCtx, order); err != nil {
			return err
		}
		return r.createOrderItems(txCtx, order)
	})
}

// createOrder вставляет запись в таблицу orders.
func (r *repository) createOrder(ctx context.Context, order model.Order) error {
	const query = `INSERT INTO orders (uuid, status, created_at) VALUES ($1, $2, $3)`

	_, err := r.getter.DefaultTrOrDB(ctx, r.pool).Exec(
		ctx,
		query,
		order.UUID,
		order.Status,
		order.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("создать заказ: %w", err)
	}

	return nil
}

// createOrderItems выполняет batch-insert строк заказа в order_items.
// order_uuid берётся из order.UUID (родитель агрегата), PartUUID/PartType/Price —
// из order.Items: в самой model.OrderItem ссылки на родителя нет.
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
		return fmt.Errorf("собрать запрос: %w", err)
	}

	_, err = r.getter.DefaultTrOrDB(ctx, r.pool).Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("записать детали заказа: %w", err)
	}

	return nil
}
