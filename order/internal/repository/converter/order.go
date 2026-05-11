package converter

import (
	"github.com/vixart/rocket-factory/order/internal/model"
	"github.com/vixart/rocket-factory/order/internal/repository/record"
)

func OrderModelToRecord(m model.Order) record.Order {
	return record.Order{
		OrderUUID:       m.OrderUUID,
		HullUUID:        m.HullUUID,
		EngineUUID:      m.EngineUUID,
		ShieldUUID:      m.ShieldUUID,
		WeaponUUID:      m.WeaponUUID,
		TotalPrice:      m.TotalPrice,
		TransactionUUID: m.TransactionUUID,
		PaymentMethod:   m.PaymentMethod,
		Status:          m.Status,
		CreatedAt:       m.CreatedAt,
	}
}

func OrderRecordToModel(r record.Order) model.Order {
	return model.Order{
		OrderUUID:       r.OrderUUID,
		HullUUID:        r.HullUUID,
		EngineUUID:      r.EngineUUID,
		ShieldUUID:      r.ShieldUUID,
		WeaponUUID:      r.WeaponUUID,
		TotalPrice:      r.TotalPrice,
		TransactionUUID: r.TransactionUUID,
		PaymentMethod:   r.PaymentMethod,
		Status:          r.Status,
		CreatedAt:       r.CreatedAt,
	}
}
