package v1

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
	authv1 "github.com/vixart/rocket-factory/shared/pkg/proto/auth/v1"
)

type Client struct {
	grpcClient authv1.AuthServiceClient
}

func New(grpcClient authv1.AuthServiceClient) *Client {
	return &Client{
		grpcClient: grpcClient,
	}
}

func (c *Client) Whoami(ctx context.Context, sessionUUID uuid.UUID) (uuid.UUID, uuid.UUID, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := c.grpcClient.Whoami(
		ctxWithTimeout,
		&authv1.WhoamiRequest{SessionUuid: sessionUUID.String()},
	)
	if err != nil {
		return uuid.Nil, uuid.Nil, mapErrors(err)
	}

	return uuid.MustParse(resp.GetUser().GetUuid()), uuid.MustParse(resp.GetSession().GetUuid()), nil
}

func mapErrors(err error) error {
	slog.Error("ошибка при обращении к iam сервису", "error", err)

	if st, ok := status.FromError(err); ok {
		var errType error
		switch st.Code() {
		case codes.Unauthenticated:
			errType = errs.ErrUnauthenticated
		case codes.InvalidArgument:
			errType = errs.ErrUnauthenticated
		default:
			errType = errs.ErrInternalError
		}

		return fmt.Errorf("обращение к сервису iam вернуло ошибку %q: %w", st.Message(), errType)
	}

	return fmt.Errorf("ошибка при обращении к iam сервису: %w", errs.ErrInternalError)
}
