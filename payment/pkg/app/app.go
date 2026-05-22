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
	// gRPC keepalive параметры.
	grpcMaxConnectionIdle     = 15 * time.Minute // Закрыть idle-соединения (нет активных RPC)
	grpcMaxConnectionAge      = 30 * time.Minute // Принудительная ротация для балансировки
	grpcMaxConnectionAgeGrace = 5 * time.Second  // Время на завершение активных RPC
	grpcKeepaliveTime         = 5 * time.Minute  // Интервал ping'ов для обнаружения мёртвых соединений
	grpcKeepaliveTimeout      = 1 * time.Second  // Тайм-аут ожидания pong
	grpcMinPingInterval       = 5 * time.Minute  // Минимальный интервал ping'ов от клиента (защита от DoS)
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
			PermitWithoutStream: true, // Разрешить "тёплые" соединения без активных RPC
		}),
		grpc.UnaryInterceptor(interceptor.ErrorInterceptor),
	}
}

func RegisterServices(grpcServer *grpc.Server) {
	service := payment.NewService()
	api := paymentApiV1.NewApi(service)
	paymentv1.RegisterPaymentServiceServer(grpcServer, api)
}
