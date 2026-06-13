package v1

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
	inventoryv1 "github.com/vixart/rocket-factory/shared/pkg/proto/inventory/v1"
)

func (a *api) CommitParts(ctx context.Context, req *inventoryv1.CommitPartsRequest) (*inventoryv1.CommitPartsResponse, error) {
	parsedUuids := make([]uuid.UUID, 0, len(req.Uuids))
	for _, uuidStr := range req.GetUuids() {
		parsedUuid, err := uuid.Parse(uuidStr)
		if err != nil {
			return nil, fmt.Errorf("неверный формат uuid: %s, %w", uuidStr, errs.ErrInvalidUUID)
		}

		parsedUuids = append(parsedUuids, parsedUuid)
	}

	err := a.inventoryService.Commit(ctx, parsedUuids)
	if err != nil {
		return nil, err
	}

	return &inventoryv1.CommitPartsResponse{}, nil
}
