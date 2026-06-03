package converter

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/vixart/rocket-factory/inventory/internal/model/entity"
	"github.com/vixart/rocket-factory/inventory/internal/model/valueobject"
	inventoryv1 "github.com/vixart/rocket-factory/shared/pkg/proto/inventory/v1"
)

func PartModelToPartProto(p entity.Part) *inventoryv1.Part {
	return &inventoryv1.Part{
		Uuid:          p.UUID().String(),
		Name:          p.Name(),
		Description:   p.Description(),
		Price:         p.Price(),
		PartType:      PartTypeModelToProto(p.PartType()),
		StockQuantity: int64(p.StockQuantity()),
		CreatedAt:     timestamppb.New(p.CreatedAt()),
	}
}

func PartTypeModelToProto(pt valueobject.PartType) inventoryv1.PartType {
	switch pt {
	case valueobject.PartTypeHull:
		return inventoryv1.PartType_PART_TYPE_HULL
	case valueobject.PartTypeEngine:
		return inventoryv1.PartType_PART_TYPE_ENGINE
	case valueobject.PartTypeShield:
		return inventoryv1.PartType_PART_TYPE_SHIELD
	case valueobject.PartTypeWeapon:
		return inventoryv1.PartType_PART_TYPE_WEAPON
	default:
		return inventoryv1.PartType_PART_TYPE_UNSPECIFIED
	}
}

func PartTypeProtoToModel(pt inventoryv1.PartType) valueobject.PartType {
	switch pt {
	case inventoryv1.PartType_PART_TYPE_HULL:
		return valueobject.PartTypeHull
	case inventoryv1.PartType_PART_TYPE_ENGINE:
		return valueobject.PartTypeEngine
	case inventoryv1.PartType_PART_TYPE_SHIELD:
		return valueobject.PartTypeShield
	case inventoryv1.PartType_PART_TYPE_WEAPON:
		return valueobject.PartTypeWeapon
	default:
		return valueobject.PartTypeUnspecified
	}
}
