package record

import (
	"time"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/order/internal/model"
)

type Order struct {
	UUID            uuid.UUID            `db:"uuid"`
	UserUUID        uuid.UUID            `db:"user_uuid"`
	Status          model.OrderStatus    `db:"status"`
	TransactionUUID *uuid.UUID           `db:"transaction_uuid"`
	PaymentMethod   *model.PaymentMethod `db:"payment_method"`
	CreatedAt       time.Time            `db:"created_at"`
}
