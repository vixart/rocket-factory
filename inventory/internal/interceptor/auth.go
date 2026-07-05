package interceptor

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/vixart/rocket-factory/platform/pkg/auth"
)

type IAMService interface {
	Whoami(ctx context.Context, sessionUUID uuid.UUID) (uuid.UUID, uuid.UUID, error)
}

func Auth(iamService IAMService) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "отсутствует metadata")
		}

		values := md.Get(auth.SessionTokenKey)
		if len(values) == 0 {
			return nil, status.Error(codes.Unauthenticated, "отсутствует session-uuid")
		}

		sessionUUID, err := uuid.Parse(values[0])
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "неверный формат session-uuid")
		}

		userUUID, sessionUUID, err := iamService.Whoami(ctx, sessionUUID)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "сессия не действительна")
		}

		ctx = auth.WithUserUUID(ctx, userUUID)
		ctx = auth.WithSessionUUID(ctx, sessionUUID)

		return handler(ctx, req)
	}
}
