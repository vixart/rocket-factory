package converter

import (
	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/order/internal/model"
	inventoryv1 "github.com/vixart/rocket-factory/shared/pkg/proto/inventory/v1"
)

func PartFromProto(part *inventoryv1.Part, partUuid uuid.UUID) model.Part {
	return model.Part{
		UUID:          partUuid,
		Name:          part.GetName(),
		Price:         part.GetPrice(),
		StockQuantity: part.GetStockQuantity(),
	}
}
