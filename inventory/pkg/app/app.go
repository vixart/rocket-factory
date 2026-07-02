package app

import (
	"time"

	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	inventoryApiV1 "github.com/vixart/rocket-factory/inventory/internal/api/inventory/v1"
	iamClient "github.com/vixart/rocket-factory/inventory/internal/client/grpc/iam/v1"
	"github.com/vixart/rocket-factory/inventory/internal/interceptor"
	partRepository "github.com/vixart/rocket-factory/inventory/internal/repository/part"
	"github.com/vixart/rocket-factory/inventory/internal/service/application/part"
	"github.com/vixart/rocket-factory/inventory/internal/service/domain"
	authv1 "github.com/vixart/rocket-factory/shared/pkg/proto/auth/v1"
	inventoryv1 "github.com/vixart/rocket-factory/shared/pkg/proto/inventory/v1"
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

func Interceptors(authClient authv1.AuthServiceClient) []grpc.ServerOption {
	client := iamClient.New(authClient)
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
		grpc.ChainUnaryInterceptor(
			interceptor.ErrorInterceptor,
			interceptor.Auth(client),
		),
	}
}

func RegisterServices(
	txManager *manager.Manager,
	grpcServer *grpc.Server,
	pool *pgxpool.Pool,
) {
	repo := partRepository.NewRepository(pool)
	service := part.NewService(txManager, repo, domain.NewCompatibilityChecker())
	api := inventoryApiV1.NewApi(service)
	inventoryv1.RegisterInventoryServiceServer(grpcServer, api)
}
