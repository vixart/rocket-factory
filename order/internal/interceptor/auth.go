package interceptor

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/vixart/rocket-factory/platform/pkg/auth"
)

func SessionForwarder() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		token, ok := auth.SessionUUIDFromContext(ctx)
		if ok {
			ctx = metadata.AppendToOutgoingContext(ctx, auth.SessionTokenKey, token)
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
