package v1

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/inventory/internal/api/converter"
	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
	inventoryv1 "github.com/vixart/rocket-factory/shared/pkg/proto/inventory/v1"
)

func (a *api) GetPart(ctx context.Context, req *inventoryv1.GetPartRequest) (*inventoryv1.GetPartResponse, error) {
	if req.GetUuid() == "" {
		return nil, fmt.Errorf("uuid не может быть пустым, %w", errs.ErrInvalidUUID)
	}

	parsedUuid, err := uuid.Parse(req.GetUuid())
	if err != nil {
		return nil, fmt.Errorf("неверный формат uuid: %s, %w", req.GetUuid(), errs.ErrInvalidUUID)
	}

	p, err := a.inventoryService.Get(ctx, parsedUuid)
	if err != nil {
		return nil, err
	}

	return &inventoryv1.GetPartResponse{
		Part: converter.PartModelToPartProto(p),
	}, nil
}
