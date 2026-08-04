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
	// gRPC keepalive parameters.
	grpcMaxConnectionIdle     = 15 * time.Minute // close idle connections (no active RPCs)
	grpcMaxConnectionAge      = 30 * time.Minute // forced rotation, helps load balancing
	grpcMaxConnectionAgeGrace = 5 * time.Second  // grace period for in-flight RPCs
	grpcKeepaliveTime         = 5 * time.Minute  // ping interval to detect dead connections
	grpcKeepaliveTimeout      = 1 * time.Second  // pong wait timeout
	grpcMinPingInterval       = 5 * time.Second  // minimum client ping interval (must be below the client keepalive.Time)
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
			PermitWithoutStream: true, // allow warm connections without active RPCs
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
