package v1

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	converter2 "github.com/vixart/rocket-factory/inventory/internal/api/converter"
	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
	inventoryv1 "github.com/vixart/rocket-factory/shared/pkg/proto/inventory/v1"
)

func (a *api) ListParts(
	ctx context.Context,
	req *inventoryv1.ListPartsRequest,
) (*inventoryv1.ListPartsResponse, error) {
	parsedUuids := make([]uuid.UUID, 0, len(req.Uuids))
	for _, uuidStr := range req.GetUuids() {
		parsedUuid, err := uuid.Parse(uuidStr)
		if err != nil {
			return nil, fmt.Errorf("invalid uuid format: %s, %w", uuidStr, errs.ErrInvalidUUID)
		}

		parsedUuids = append(parsedUuids, parsedUuid)
	}

	parts, err := a.inventoryService.List(ctx, parsedUuids, converter2.PartTypeProtoToModel(req.GetPartType()))
	if err != nil {
		return nil, err
	}

	var partsProto []*inventoryv1.Part
	for _, part := range parts {
		partsProto = append(partsProto, converter2.PartModelToPartProto(part))
	}

	return &inventoryv1.ListPartsResponse{Parts: partsProto}, nil
}
