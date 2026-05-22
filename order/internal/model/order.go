package model

import (
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	OrderStatusPendingPayment OrderStatus = "PENDING_PAYMENT"
	OrderStatusPaid           OrderStatus = "PAID"
	OrderStatusCancelled      OrderStatus = "CANCELLED"
)

type OrderParts struct {
	HullUUID   uuid.UUID
	EngineUUID uuid.UUID
	ShieldUUID *uuid.UUID
	WeaponUUID *uuid.UUID
}

type OrderItem struct {
	UUID     uuid.UUID
	PartType PartType
	Price    int64 // в копейках
}

type Order struct {
	UUID            uuid.UUID
	Items           []OrderItem
	TransactionUUID *uuid.UUID
	PaymentMethod   *PaymentMethod
	Status          OrderStatus
	CreatedAt       time.Time
}

func (o Order) TotalPrice() int64 {
	var total int64

	for _, item := range o.Items {
		total += item.Price
	}

	return total
}
