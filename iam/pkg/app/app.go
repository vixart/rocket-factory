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
	// gRPC keepalive parameters.
	grpcMaxConnectionIdle     = 15 * time.Minute // close idle connections (no active RPCs)
	grpcMaxConnectionAge      = 30 * time.Minute // forced rotation, helps load balancing
	grpcMaxConnectionAgeGrace = 5 * time.Second  // grace period for in-flight RPCs
	grpcKeepaliveTime         = 5 * time.Minute  // ping interval to detect dead connections
	grpcKeepaliveTimeout      = 1 * time.Second  // pong wait timeout
	grpcMinPingInterval       = 5 * time.Second  // minimum client ping interval (must be below the client keepalive.Time)
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
			PermitWithoutStream: true, // clients ping without active RPCs, otherwise the connection is dropped
		}),
		grpc.UnaryInterceptor(interceptor.ErrorInterceptor),
	)

	userv1.RegisterUserServiceServer(grpcServer, userApi)
	authv1.RegisterAuthServiceServer(grpcServer, authApi)

	return grpcServer
}
