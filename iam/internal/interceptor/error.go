package interceptor

import (
	"context"

	"github.com/go-faster/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	errs "github.com/vixart/rocket-factory/iam/internal/errors"
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
	case errors.Is(err, errs.ErrUserNotFound):
		return nil, status.Error(codes.NotFound, err.Error())

	case errors.Is(err, errs.ErrUserAlreadyExists):
		return nil, status.Error(codes.AlreadyExists, err.Error())

	case errors.Is(err, errs.ErrInvalidCredentials):
		return nil, status.Error(codes.Unauthenticated, err.Error())

	case errors.Is(err, errs.ErrSessionNotFound):
		return nil, status.Error(codes.Unauthenticated, err.Error())

	case errors.Is(err, errs.ErrInvalidLogin):
		return nil, status.Error(codes.InvalidArgument, err.Error())

	case errors.Is(err, errs.ErrWeakPassword):
		return nil, status.Error(codes.InvalidArgument, err.Error())

	case errors.Is(err, errs.ErrEmptyCredential):
		return nil, status.Error(codes.InvalidArgument, err.Error())

	case errors.Is(err, errs.ErrEmptySessionID):
		return nil, status.Error(codes.InvalidArgument, err.Error())

	case errors.Is(err, errs.ErrInvalidUUID):
		return nil, status.Error(codes.InvalidArgument, err.Error())

	default:
		return nil, status.Error(codes.Internal, "внутренняя ошибка")
	}
}
