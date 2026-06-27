package app

import (
	"context"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	apiAuthDeps "github.com/vixart/rocket-factory/iam/internal/api/auth/v1"
	apiUserDeps "github.com/vixart/rocket-factory/iam/internal/api/user/v1"
	"github.com/vixart/rocket-factory/iam/internal/config"
	sessionRepoDeps "github.com/vixart/rocket-factory/iam/internal/repository/session"
	userRepoDeps "github.com/vixart/rocket-factory/iam/internal/repository/user"
	iamServiceDeps "github.com/vixart/rocket-factory/iam/internal/service/iam"
	"github.com/vixart/rocket-factory/platform/pkg/closer"
	redisPlatform "github.com/vixart/rocket-factory/platform/pkg/redis"
	authv1 "github.com/vixart/rocket-factory/shared/pkg/proto/auth/v1"
	userv1 "github.com/vixart/rocket-factory/shared/pkg/proto/user/v1"
)

type diContainer struct {
	pgPool      *pgxpool.Pool
	redisClient *redis.Client

	userRepo    iamServiceDeps.UserRepository
	sessionRepo iamServiceDeps.SessionRepository

	userSvc    apiUserDeps.UserService
	sessionSvc apiAuthDeps.SessionService

	userV1Handler userv1.UserServiceServer
	authV1Handler authv1.AuthServiceServer
}

func (d *diContainer) PGPool(ctx context.Context) *pgxpool.Pool {
	if d.pgPool == nil {
		pool, err := pgxpool.New(ctx, config.AppConfig().PG.DSN())
		if err != nil {
			slog.Error("не удалось подключиться к PostgreSQL", "error", err)
			os.Exit(1)
		}

		err = pool.Ping(ctx)
		if err != nil {
			slog.Error("не удалось выполнить ping PostgreSQL", "error", err)
			os.Exit(1)
		}

		closer.Add("PostgreSQL pool", func(_ context.Context) error {
			pool.Close()
			return nil
		})

		d.pgPool = pool
	}

	return d.pgPool
}

func (d *diContainer) RedisClient(_ context.Context) *redis.Client {
	if d.redisClient == nil {
		rdb, err := redisPlatform.NewClient(&redis.Options{
			Addr:            config.AppConfig().Redis.Address(),
			DialTimeout:     config.AppConfig().Redis.ConnectionTimeout,
			ReadTimeout:     config.AppConfig().Redis.ConnectionTimeout,
			WriteTimeout:    config.AppConfig().Redis.ConnectionTimeout,
			MaxIdleConns:    config.AppConfig().Redis.MaxIdle,
			ConnMaxIdleTime: config.AppConfig().Redis.IdleTimeout,
		}, slog.Default())
		if err != nil {
			slog.Error("не удалось создать Redis клиент", "error", err)
			os.Exit(1)
		}

		closer.Add("Redis", func(_ context.Context) error {
			return rdb.Close()
		})

		d.redisClient = rdb
	}

	return d.redisClient
}

func (d *diContainer) UserRepo(ctx context.Context) iamServiceDeps.UserRepository {
	if d.userRepo == nil {
		d.userRepo = userRepoDeps.NewRepository(d.PGPool(ctx))
	}

	return d.userRepo
}

func (d *diContainer) SessionRepo(ctx context.Context) iamServiceDeps.SessionRepository {
	if d.sessionRepo == nil {
		d.sessionRepo = sessionRepoDeps.NewRepository(d.RedisClient(ctx))
	}

	return d.sessionRepo
}

func (d *diContainer) UserSvc(ctx context.Context) apiUserDeps.UserService {
	if d.userSvc == nil {
		d.userSvc = iamServiceDeps.NewService(
			d.UserRepo(ctx),
			d.SessionRepo(ctx),
			config.AppConfig().Session.TTL,
			bcrypt.DefaultCost,
		)
	}

	return d.userSvc
}

func (d *diContainer) SessionSvc(ctx context.Context) apiAuthDeps.SessionService {
	if d.sessionSvc == nil {
		d.sessionSvc = iamServiceDeps.NewService(
			d.UserRepo(ctx),
			d.SessionRepo(ctx),
			config.AppConfig().Session.TTL,
			bcrypt.DefaultCost,
		)
	}

	return d.sessionSvc
}

func (d *diContainer) UserV1API(ctx context.Context) userv1.UserServiceServer {
	if d.userV1Handler == nil {
		d.userV1Handler = apiUserDeps.NewApi(d.UserSvc(ctx))
	}

	return d.userV1Handler
}

func (d *diContainer) AuthV1API(ctx context.Context) authv1.AuthServiceServer {
	if d.authV1Handler == nil {
		d.authV1Handler = apiAuthDeps.NewApi(d.SessionSvc(ctx))
	}

	return d.authV1Handler
}
