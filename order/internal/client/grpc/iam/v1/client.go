package v1

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	errs "github.com/vixart/rocket-factory/order/internal/errors"
	authv1 "github.com/vixart/rocket-factory/shared/pkg/proto/auth/v1"
)

type client struct {
	grpcClient authv1.AuthServiceClient
}

func NewClient(grpcClient authv1.AuthServiceClient) *client {
	return &client{
		grpcClient: grpcClient,
	}
}

func (c *client) Whoami(ctx context.Context, sessionUUID uuid.UUID) (uuid.UUID, uuid.UUID, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := c.grpcClient.Whoami(
		ctxWithTimeout,
		&authv1.WhoamiRequest{SessionUuid: sessionUUID.String()},
	)
	if err != nil {
		return uuid.Nil, uuid.Nil, mapErrors(ctx, err)
	}

	return uuid.MustParse(resp.GetUser().GetUuid()), uuid.MustParse(resp.GetSession().GetUuid()), nil
}

func mapErrors(ctx context.Context, err error) error {
	if st, ok := status.FromError(err); ok {
		var errType error
		switch st.Code() {
		case codes.Unauthenticated, codes.InvalidArgument:
			errType = errs.ErrUnauthorized
			slog.WarnContext(ctx, "iam отклонил запрос", "code", st.Code().String(), "error", err)
		default:
			errType = errs.ErrInternalError
			slog.ErrorContext(ctx, "ошибка при обращении к iam сервису",
				"code", st.Code().String(), "error", err)
		}

		return fmt.Errorf("обращение к сервису iam вернуло ошибку %q: %w", st.Message(), errType)
	}

	slog.ErrorContext(ctx, "ошибка при обращении к iam сервису", "error", err)

	return fmt.Errorf("ошибка при обращении к iam сервису: %w", errs.ErrInternalError)
}
