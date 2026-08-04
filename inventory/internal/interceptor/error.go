package interceptor

import (
	"context"

	"github.com/go-faster/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
)

func ErrorInterceptor(
	ctx context.Context,
	req any,
	_ *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	resp, err := handler(ctx, req)
	if err == nil {
		return resp, nil
	}

	switch {
	case errors.Is(err, errs.ErrPartNotFound):
		return nil, status.Error(codes.NotFound, err.Error())

	case errors.Is(err, errs.ErrInvalidUUID),
		errors.Is(err, errs.ErrPartTypeMismatch):
		return nil, status.Error(codes.InvalidArgument, err.Error())

	case errors.Is(err, errs.ErrIncompatibleParts),
		errors.Is(err, errs.ErrNothingToCommit),
		errors.Is(err, errs.ErrNothingToRelease):
		return nil, status.Error(codes.FailedPrecondition, err.Error())

	case errors.Is(err, errs.ErrOutOfStock):
		return nil, status.Error(codes.ResourceExhausted, err.Error())

	case errors.Is(err, errs.ErrInvalidProperties):
		return nil, status.Error(codes.Internal, err.Error())

	case status.Code(err) != codes.OK:
		return nil, err

	default:
		return nil, status.Error(codes.Internal, "internal error")
	}
}
