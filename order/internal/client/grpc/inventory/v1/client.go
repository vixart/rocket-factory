package v1

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/vixart/rocket-factory/order/internal/client/grpc/inventory/v1/converter"
	errs "github.com/vixart/rocket-factory/order/internal/errors"
	"github.com/vixart/rocket-factory/order/internal/model"
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

func (c *client) ListParts(ctx context.Context, uuids []uuid.UUID) (map[uuid.UUID]*model.Part, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := c.grpcClient.ListParts(
		ctxWithTimeout,
		&inventoryv1.ListPartsRequest{Uuids: uuidsToStrings(uuids)},
	)
	if err != nil {
		return nil, mapErrors(err)
	}

	parts := resp.GetParts()

	result := make(map[uuid.UUID]*model.Part, len(parts))

	for _, part := range parts {
		parsedUuid, err := uuid.Parse(part.GetUuid())
		if err != nil {
			return nil, mapErrors(err)
		}

		result[parsedUuid] = new(converter.PartFromProto(part, parsedUuid))
	}

	return result, nil
}

func uuidsToStrings(uuids []uuid.UUID) []string {
	uuidsStrings := make([]string, 0, len(uuids))

	for _, u := range uuids {
		uuidsStrings = append(uuidsStrings, u.String())
	}

	return uuidsStrings
}

func mapErrors(err error) error {
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.NotFound:
			return fmt.Errorf("деталь не найдена в сервисе inventory: %w", errs.ErrInventoryPartNotFound)
		case codes.InvalidArgument:
			return fmt.Errorf("в сервис inventory был передан неверный uuid детали: %w", errs.ErrInvalidUUID)
		default:
			return fmt.Errorf("ошибка при обращении к inventory сервису: %s | %w", st.Message(), errs.ErrInternalError)
		}
	}

	return fmt.Errorf("ошибка при обращении к inventory сервису: %w", errs.ErrInternalError)
}
