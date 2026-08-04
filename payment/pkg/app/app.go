package app

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	paymentApiV1 "github.com/vixart/rocket-factory/payment/internal/api/payment/v1"
	"github.com/vixart/rocket-factory/payment/internal/interceptor"
	"github.com/vixart/rocket-factory/payment/internal/service/payment"
	paymentv1 "github.com/vixart/rocket-factory/shared/pkg/proto/payment/v1"
)

const (
	// gRPC keepalive parameters.
	grpcMaxConnectionIdle     = 15 * time.Minute // close idle connections (no active RPCs)
	grpcMaxConnectionAge      = 30 * time.Minute // forced rotation, helps load balancing
	grpcMaxConnectionAgeGrace = 5 * time.Second  // grace period for in-flight RPCs
	grpcKeepaliveTime         = 5 * time.Minute  // ping interval to detect dead connections
	grpcKeepaliveTimeout      = 1 * time.Second  // pong wait timeout
	grpcMinPingInterval       = 5 * time.Second  // minimum client ping interval (must be below the client keepalive.Time)
)

func Interceptors() []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     grpcMaxConnectionIdle,
			MaxConnectionAge:      grpcMaxConnectionAge,
			MaxConnectionAgeGrace: grpcMaxConnectionAgeGrace,
			Time:                  grpcKeepaliveTime,
			Timeout:               grpcKeepaliveTimeout,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             grpcMinPingInterval,
			PermitWithoutStream: true, // allow warm connections without active RPCs
		}),
		grpc.UnaryInterceptor(interceptor.ErrorInterceptor),
	}
}

func RegisterServices(grpcServer *grpc.Server) {
	service := payment.NewService()
	api := paymentApiV1.NewApi(service)
	paymentv1.RegisterPaymentServiceServer(grpcServer, api)
}
