package app

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	apiAuthDeps "github.com/vixart/rocket-factory/iam/internal/api/auth/v1"
	apiUserDeps "github.com/vixart/rocket-factory/iam/internal/api/user/v1"
	"github.com/vixart/rocket-factory/iam/internal/interceptor"
	sessionRepoDeps "github.com/vixart/rocket-factory/iam/internal/repository/session"
	userRepoDeps "github.com/vixart/rocket-factory/iam/internal/repository/user"
	iamServiceDeps "github.com/vixart/rocket-factory/iam/internal/service/iam"
	authv1 "github.com/vixart/rocket-factory/shared/pkg/proto/auth/v1"
	userv1 "github.com/vixart/rocket-factory/shared/pkg/proto/user/v1"
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

func NewGRPCServer(
	pool *pgxpool.Pool,
	rdb *redis.Client,
	sessionTTL time.Duration,
	bcryptCost int,
) *grpc.Server {
	userRepo := userRepoDeps.NewRepository(pool)
	sessionRepo := sessionRepoDeps.NewRepository(rdb)

	userSvc := iamServiceDeps.NewService(userRepo, sessionRepo, sessionTTL, bcryptCost)
	authSvc := iamServiceDeps.NewService(userRepo, sessionRepo, sessionTTL, bcryptCost)

	userApi := apiUserDeps.NewApi(userSvc)
	authApi := apiAuthDeps.NewApi(authSvc)

	grpcServer := grpc.NewServer(
		grpc.Creds(insecure.NewCredentials()),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     grpcMaxConnectionIdle,
			MaxConnectionAge:      grpcMaxConnectionAge,
			MaxConnectionAgeGrace: grpcMaxConnectionAgeGrace,
			Time:                  grpcKeepaliveTime,
			Timeout:               grpcKeepaliveTimeout,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             grpcMinPingInterval,
			PermitWithoutStream: false,
		}),
		grpc.UnaryInterceptor(interceptor.ErrorInterceptor),
	)

	userv1.RegisterUserServiceServer(grpcServer, userApi)
	authv1.RegisterAuthServiceServer(grpcServer, authApi)

	return grpcServer
}
