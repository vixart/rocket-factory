package record

import (
	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/order/internal/model"
)

type OrderItem struct {
	OrderUUID uuid.UUID      `db:"order_uuid"`
	PartUUID  uuid.UUID      `db:"part_uuid"`
	PartType  model.PartType `db:"part_type"`
	Price     int64          `db:"price"` // в копейках
}
