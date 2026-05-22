package order

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	errs "github.com/vixart/rocket-factory/order/internal/errors"
	"github.com/vixart/rocket-factory/order/internal/model"
	"github.com/vixart/rocket-factory/order/internal/repository/converter"
	"github.com/vixart/rocket-factory/order/internal/repository/record"
)

func (r *repository) Get(ctx context.Context, uuid uuid.UUID) (model.Order, error) {
	order, err := r.getOrder(ctx, uuid)
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

func (r *repository) getOrder(ctx context.Context, uuid uuid.UUID) (model.Order, error) {
	const queryOrder = `SELECT uuid, status, transaction_uuid, payment_method, created_at 
							FROM orders WHERE uuid = $1`

	var order record.Order

	err := r.getter.DefaultTrOrDB(ctx, r.pool).QueryRow(ctx, queryOrder, uuid).Scan(
		&order.UUID, &order.Status, &order.TransactionUUID, &order.PaymentMethod, &order.CreatedAt)
	if err != nil {
		return model.Order{}, err
	}

	return converter.OrderRecordToModel(order), nil
}

func (r *repository) getOrderItems(ctx context.Context, uuid uuid.UUID) ([]model.OrderItem, error) {
	const queryOrderItems = `SELECT order_uuid, part_uuid, part_type, price FROM order_items WHERE order_uuid = $1`

	rows, err := r.getter.DefaultTrOrDB(ctx, r.pool).Query(ctx, queryOrderItems, uuid)
	if err != nil {
		return []model.OrderItem{}, fmt.Errorf("получить список деталей: %w", err)
	}

	defer rows.Close()

	orderItemsRecords, err := pgx.CollectRows(rows, pgx.RowToStructByName[record.OrderItem])
	if err != nil {
		return []model.OrderItem{}, err
	}

	orderItemsModels := converter.OrderItemRecordsToModels(orderItemsRecords)

	return orderItemsModels, nil
}
