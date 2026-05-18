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

func (a *api) ListParts(
	ctx context.Context,
	req *inventoryv1.ListPartsRequest,
) (*inventoryv1.ListPartsResponse, error) {
	parsedUuids := make([]uuid.UUID, 0, len(req.Uuids))
	for _, uuidStr := range req.Uuids {
		parsedUuid, err := uuid.Parse(uuidStr)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "неверный формат uuid: %s", uuidStr)
		}

		parsedUuids = append(parsedUuids, parsedUuid)
	}

	parts, err := a.inventoryService.List(ctx, parsedUuids, converter.PartTypeProtoToModel(req.GetPartType()))
	if errors.Is(err, errs.ErrPartNotFound) {
		return nil, status.Errorf(codes.NotFound, "%s", err.Error())
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "что-то пошло не так: %s", err)
	}

	var partsProto []*inventoryv1.Part
	for _, part := range parts {
		partsProto = append(partsProto, converter.PartModelToPartProto(part))
	}

	return &inventoryv1.ListPartsResponse{Parts: partsProto}, nil
}
