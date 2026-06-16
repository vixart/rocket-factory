package order

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/vixart/rocket-factory/order/internal/model"
	"github.com/vixart/rocket-factory/order/internal/repository/converter"
	"github.com/vixart/rocket-factory/order/internal/repository/record"
)

func (r *repository) getOrder(ctx context.Context, uuid uuid.UUID) (model.Order, error) {
	const queryOrder = `SELECT uuid, user_uuid, status, transaction_uuid, payment_method, created_at 
							FROM orders WHERE uuid = $1`

	var order record.Order

	err := r.getter.DefaultTrOrDB(ctx, r.pool).QueryRow(ctx, queryOrder, uuid).Scan(
		&order.UUID, &order.UserUUID, &order.Status, &order.TransactionUUID, &order.PaymentMethod, &order.CreatedAt)
	if err != nil {
		return model.Order{}, err
	}

	return converter.OrderRecordToModel(order), nil
}

func (r *repository) getOrderForUpdate(ctx context.Context, uuid uuid.UUID) (model.Order, error) {
	const queryOrder = `SELECT uuid, user_uuid, status, transaction_uuid, payment_method, created_at 
							FROM orders WHERE uuid = $1 FOR UPDATE`

	var order record.Order

	err := r.getter.DefaultTrOrDB(ctx, r.pool).QueryRow(ctx, queryOrder, uuid).Scan(
		&order.UUID, &order.UserUUID, &order.Status, &order.TransactionUUID, &order.PaymentMethod, &order.CreatedAt)
	if err != nil {
		return model.Order{}, err
	}

	return converter.OrderRecordToModel(order), nil
}

func (r *repository) getOrderItems(ctx context.Context, uuid uuid.UUID) ([]model.OrderItem, error) {
	const queryOrderItems = `SELECT order_uuid, part_uuid, part_type, price FROM order_items WHERE order_uuid = $1`

	rows, err := r.getter.DefaultTrOrDB(ctx, r.pool).Query(ctx, queryOrderItems, uuid)
	if err != nil {
		return nil, fmt.Errorf("получить список деталей: %w", err)
	}

	defer rows.Close()

	orderItemsRecords, err := pgx.CollectRows(rows, pgx.RowToStructByName[record.OrderItem])
	if err != nil {
		return nil, err
	}

	return converter.OrderItemRecordsToModels(orderItemsRecords), nil
}
