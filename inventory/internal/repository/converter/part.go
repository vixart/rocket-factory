package converter

import (
	"github.com/vixart/rocket-factory/inventory/internal/model"
	"github.com/vixart/rocket-factory/inventory/internal/repository/record"
)

func PartRecordToModel(p record.Part) model.Part {
	return model.Part{
		UUID:          p.UUID,
		Name:          p.Name,
		Description:   p.Description,
		Price:         p.Price,
		PartType:      p.PartType,
		StockQuantity: p.StockQuantity,
		CreatedAt:     p.CreatedAt,
	}
}
