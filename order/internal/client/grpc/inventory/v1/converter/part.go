package converter

import (
	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/order/internal/model"
	inventoryv1 "github.com/vixart/rocket-factory/shared/pkg/proto/inventory/v1"
)

func PartFromProto(part *inventoryv1.Part, partUuid uuid.UUID) model.Part {
	return model.Part{
		UUID:          partUuid,
		PartType:      PartTypeFromProto(part.GetPartType()),
		Name:          part.GetName(),
		Price:         part.GetPrice(),
		StockQuantity: part.GetStockQuantity(),
	}
}

func UuidsToStrings(uuids []uuid.UUID) []string {
	uuidsStrings := make([]string, 0, len(uuids))

	for _, u := range uuids {
		uuidsStrings = append(uuidsStrings, u.String())
	}

	return uuidsStrings
}

func PartTypeFromProto(partType inventoryv1.PartType) model.PartType {
	switch partType {
	case inventoryv1.PartType_PART_TYPE_HULL:
		return model.PartTypeHull
	case inventoryv1.PartType_PART_TYPE_ENGINE:
		return model.PartTypeEngine
	case inventoryv1.PartType_PART_TYPE_SHIELD:
		return model.PartTypeShield
	case inventoryv1.PartType_PART_TYPE_WEAPON:
		return model.PartTypeWeapon
	default:
		return model.PartTypeUnspecified
	}
}
