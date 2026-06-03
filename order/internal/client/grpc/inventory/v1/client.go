package v1

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/vixart/rocket-factory/order/internal/client/grpc/inventory/v1/converter"
	errs "github.com/vixart/rocket-factory/order/internal/errors"
	"github.com/vixart/rocket-factory/order/internal/model"
	"github.com/vixart/rocket-factory/order/internal/service/input"
	"github.com/vixart/rocket-factory/platform/pkg/util"
	inventoryv1 "github.com/vixart/rocket-factory/shared/pkg/proto/inventory/v1"
)

type client struct {
	grpcClient inventoryv1.InventoryServiceClient
}

func NewClient(grpcClient inventoryv1.InventoryServiceClient) *client {
	return &client{
		grpcClient: grpcClient,
	}
}

func (c *client) ListParts(ctx context.Context, uuids []uuid.UUID) ([]model.Part, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := c.grpcClient.ListParts(
		ctxWithTimeout,
		&inventoryv1.ListPartsRequest{Uuids: converter.UuidsToStrings(uuids)},
	)
	if err != nil {
		return nil, mapErrors(err)
	}

	parts := resp.GetParts()

	result := make([]model.Part, 0, len(parts))

	for _, part := range parts {
		parsedUuid, err := uuid.Parse(part.GetUuid())
		if err != nil {
			return nil, mapErrors(err)
		}

		result = append(result, converter.PartFromProto(part, parsedUuid))
	}

	return result, nil
}

func (c *client) ReserveParts(ctx context.Context, uuids []uuid.UUID) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.grpcClient.ReserveParts(
		ctxWithTimeout,
		&inventoryv1.ReservePartsRequest{Uuids: converter.UuidsToStrings(uuids)},
	)
	if err != nil {
		return mapErrors(err)
	}

	return nil
}

func (c *client) ReleaseParts(ctx context.Context, uuids []uuid.UUID) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.grpcClient.ReleaseParts(
		ctxWithTimeout,
		&inventoryv1.ReleasePartsRequest{Uuids: converter.UuidsToStrings(uuids)},
	)
	if err != nil {
		return mapErrors(err)
	}

	return nil
}

func (c *client) ValidateCompatibility(ctx context.Context, orderParts input.OrderParts) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.grpcClient.ValidateCompatibility(
		ctxWithTimeout,
		&inventoryv1.ValidateCompatibilityRequest{
			HullUuid:   orderParts.HullUUID.String(),
			EngineUuid: orderParts.EngineUUID.String(),
			ShieldUuid: util.UUIDPtrToString(orderParts.ShieldUUID),
			WeaponUuid: util.UUIDPtrToString(orderParts.WeaponUUID),
		},
	)
	if err != nil {
		return mapErrors(err)
	}

	return nil
}

func mapErrors(err error) error {
	slog.Error("ошибка при обращении к inventory сервису", "error", err)

	if st, ok := status.FromError(err); ok {
		var errType error
		switch st.Code() {
		case codes.NotFound:
			errType = errs.ErrPartNotFound
		case codes.InvalidArgument:
			errType = errs.ErrInvalidUUID
		case codes.FailedPrecondition:
			errType = errs.ErrIncompatibleParts
		case codes.ResourceExhausted:
			errType = errs.ErrOutOfStock
		default:
			errType = errs.ErrInternalError
		}

		return fmt.Errorf("обращение к сервису inventory вернуло ошибку %q: %w", st.Message(), errType)
	}

	return fmt.Errorf("ошибка при обращении к inventory сервису: %w", errs.ErrInternalError)
}
