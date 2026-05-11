package order

import (
	"sync"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/order/internal/repository/record"
)

type repository struct {
	mu     sync.RWMutex
	orders map[uuid.UUID]record.Order
}

func NewRepository() *repository { // это репо
	return &repository{
		orders: make(map[uuid.UUID]record.Order),
	}
}
