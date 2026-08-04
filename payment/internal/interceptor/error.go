package interceptor

import (
	"context"

	"github.com/go-faster/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	errs "github.com/vixart/rocket-factory/payment/internal/errors"
)

func ErrorInterceptor(
	ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
) (any, error) {
	resp, err := handler(ctx, req)
	if err == nil {
		return resp, nil
	}
	switch {
	case errors.Is(err, errs.ErrPaymentMethodNotSpecified), errors.Is(err, errs.ErrInvalidUUID):
		return nil, status.Error(codes.InvalidArgument, err.Error())
	default:
		return nil, status.Error(codes.Internal, "internal error")
	}
}
