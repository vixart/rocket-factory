package model

import (
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	OrderStatusPendingPayment OrderStatus = "pending_payment"
	OrderStatusPaid           OrderStatus = "paid"
	OrderStatusCanceled       OrderStatus = "canceled"
)

type OrderParts struct {
	HullUUID   uuid.UUID
	EngineUUID uuid.UUID
	ShieldUUID *uuid.UUID
	WeaponUUID *uuid.UUID
}

type Order struct {
	OrderUUID       uuid.UUID
	HullUUID        uuid.UUID
	EngineUUID      uuid.UUID
	ShieldUUID      *uuid.UUID // опциональный
	WeaponUUID      *uuid.UUID // опциональный
	TotalPrice      int64      // в копейках
	TransactionUUID *uuid.UUID
	PaymentMethod   *PaymentMethod
	Status          OrderStatus
	CreatedAt       time.Time
}
