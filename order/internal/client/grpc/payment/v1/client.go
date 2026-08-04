package v1

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/vixart/rocket-factory/order/internal/client/grpc/payment/v1/converter"
	errs "github.com/vixart/rocket-factory/order/internal/errors"
	"github.com/vixart/rocket-factory/order/internal/model"
	paymentv1 "github.com/vixart/rocket-factory/shared/pkg/proto/payment/v1"
)

type client struct {
	grpcClient paymentv1.PaymentServiceClient
}

func NewClient(grpcClient paymentv1.PaymentServiceClient) *client {
	return &client{
		grpcClient: grpcClient,
	}
}

func (c *client) PayOrder(
	ctx context.Context,
	orderUuid uuid.UUID,
	paymentMethod model.PaymentMethod,
) (uuid.UUID, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := c.grpcClient.PayOrder(ctxWithTimeout, &paymentv1.PayOrderRequest{
		OrderUuid:     orderUuid.String(),
		PaymentMethod: converter.MapPaymentMethod(paymentMethod),
	})
	if err != nil {
		return uuid.Nil, mapErrors(err)
	}

	txUuid, err := uuid.Parse(resp.TransactionUuid)
	if err != nil {
		return uuid.Nil, fmt.Errorf("payment service returned an invalid uuid: %w", errs.ErrInvalidUUID)
	}

	return txUuid, nil
}

func mapErrors(err error) error {
	slog.Error("payment service call failed", "error", err)
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.InvalidArgument:
			return fmt.Errorf("payment service received invalid arguments: %s | %w", st.Message(), errs.ErrPaymentFailed)
		default:
			return fmt.Errorf("payment service call failed: %s | %w", st.Message(), errs.ErrInternalError)
		}
	}

	return fmt.Errorf("payment service call failed: %w", errs.ErrInternalError)
}
