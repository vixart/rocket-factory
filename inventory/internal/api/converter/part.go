package converter

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/vixart/rocket-factory/inventory/internal/model"
	inventoryv1 "github.com/vixart/rocket-factory/shared/pkg/proto/inventory/v1"
)

func PartModelToPartProto(p model.Part) *inventoryv1.Part {
	var createdAt *timestamppb.Timestamp

	if p.CreatedAt != nil {
		createdAt = timestamppb.New(*p.CreatedAt)
	}

	return &inventoryv1.Part{
		Uuid:          p.UUID.String(),
		Name:          p.Name,
		Description:   p.Description,
		Price:         p.Price,
		PartType:      PartTypeModelToProto(p.PartType),
		StockQuantity: p.StockQuantity,
		CreatedAt:     createdAt,
	}
}
