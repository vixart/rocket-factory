package v1

import (
	"context"

	"github.com/go-faster/errors"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/vixart/rocket-factory/inventory/internal/api/inventory/converter"
	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
	inventoryv1 "github.com/vixart/rocket-factory/shared/pkg/proto/inventory/v1"
)

func (a *api) GetPart(ctx context.Context, req *inventoryv1.GetPartRequest) (*inventoryv1.GetPartResponse, error) {
	if req.GetUuid() == "" {
		return nil, status.Error(codes.InvalidArgument, "uuid не может быть пустым")
	}

	parsedUuid, err := uuid.Parse(req.GetUuid())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "неверный формат uuid: %s", req.GetUuid())
	}

	p, err := a.inventoryService.Get(ctx, parsedUuid)
	if errors.Is(errs.ErrPartNotFound, err) {
		return nil, status.Errorf(codes.NotFound, "деталь не найдена по uuid: %s", parsedUuid)
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "что-то пошло не так: %s", err)
	}

	return &inventoryv1.GetPartResponse{
		Part: converter.PartModelToPartProto(p),
	}, nil
}
