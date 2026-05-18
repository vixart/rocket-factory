package part

import (
	"sync"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/inventory/internal/repository/record"
)

type repository struct {
	mu   sync.RWMutex
	data map[uuid.UUID]record.Part
}

func NewRepository(parts *map[uuid.UUID]record.Part) *repository {
	if parts == nil {
		parts = new(make(map[uuid.UUID]record.Part))
	}
	return &repository{
		data: *parts,
	}
}
