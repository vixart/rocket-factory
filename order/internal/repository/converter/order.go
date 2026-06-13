package converter

import (
	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/order/internal/model"
	"github.com/vixart/rocket-factory/order/internal/repository/record"
)

func OrderModelToRecord(m model.Order) record.Order {
	return record.Order{
		UUID:            m.UUID,
		UserUUID:        m.UserUUID,
		TransactionUUID: m.TransactionUUID,
		PaymentMethod:   m.PaymentMethod,
		Status:          m.Status,
		CreatedAt:       m.CreatedAt,
	}
}

func OrderRecordToModel(r record.Order) model.Order {
	return model.Order{
		UUID:            r.UUID,
		UserUUID:        r.UserUUID,
		TransactionUUID: r.TransactionUUID,
		PaymentMethod:   r.PaymentMethod,
		Status:          r.Status,
		CreatedAt:       r.CreatedAt,
	}
}

func OrderItemModelToRecord(orderUuid uuid.UUID, orderItem model.OrderItem) record.OrderItem {
	return record.OrderItem{
		OrderUUID: orderUuid,
		PartUUID:  orderItem.UUID,
		PartType:  orderItem.PartType,
		Price:     orderItem.Price,
	}
}

func OrderItemRecordToModel(orderItem record.OrderItem) model.OrderItem {
	return model.OrderItem{
		UUID:     orderItem.PartUUID,
		PartType: orderItem.PartType,
		Price:    orderItem.Price,
	}
}

func OrderItemModelsToRecords(orderUuid uuid.UUID, orderItems []model.OrderItem) []record.OrderItem {
	orderRecords := make([]record.OrderItem, 0, len(orderItems))
	for _, orderItem := range orderItems {
		orderRecords = append(orderRecords, OrderItemModelToRecord(orderUuid, orderItem))
	}

	return orderRecords
}

func OrderItemRecordsToModels(orderItemRecords []record.OrderItem) []model.OrderItem {
	orderItemModels := make([]model.OrderItem, 0, len(orderItemRecords))
	for _, orderItemRecord := range orderItemRecords {
		orderItemModels = append(orderItemModels, OrderItemRecordToModel(orderItemRecord))
	}

	return orderItemModels
}
