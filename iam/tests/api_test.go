//go:build apitest

// Package tests holds the integration API tests of the IAM service.
//
// It starts PostgreSQL and Redis via testcontainers-go and runs the IAM gRPC server
// on bufconn — no real ports and no Docker Compose.
package tests

import (
	"context"
	"database/sql"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	iamApp "github.com/vixart/rocket-factory/iam/pkg/app"
	authv1 "github.com/vixart/rocket-factory/shared/pkg/proto/auth/v1"
	commonv1 "github.com/vixart/rocket-factory/shared/pkg/proto/common/v1"
	userv1 "github.com/vixart/rocket-factory/shared/pkg/proto/user/v1"
)

const bufSize = 1024 * 1024

var (
	authClient userv1.UserServiceClient
	authSvc    authv1.AuthServiceClient
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	pgContainer, dsn, err := startPostgres(ctx)
	if err != nil {
		panic("failed to start PostgreSQL: " + err.Error())
	}
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	redisContainer, redisAddr, err := startRedis(ctx)
	if err != nil {
		panic("failed to start Redis: " + err.Error())
	}
	defer func() {
		_ = redisContainer.Terminate(ctx)
	}()

	if err = applyMigrations(dsn); err != nil {
		panic("failed to apply migrations: " + err.Error())
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		panic("failed to create pgxpool: " + err.Error())
	}
	defer pool.Close()

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	defer func() {
		_ = rdb.Close()
	}()

	server := iamApp.NewGRPCServer(pool, rdb, time.Hour, bcrypt.MinCost)

	lis := bufconn.Listen(bufSize)
	go func() {
		_ = server.Serve(lis)
	}()
	defer server.Stop()

	conn, err := grpc.NewClient(
		"passthrough://bufnet",
		grpc.WithContextDialer(func(_ context.Context, _ string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic("failed to connect to bufconn: " + err.Error())
	}
	defer func() {
		_ = conn.Close()
	}()

	authClient = userv1.NewUserServiceClient(conn)
	authSvc = authv1.NewAuthServiceClient(conn)

	os.Exit(m.Run())
}

func startPostgres(ctx context.Context) (*tcpostgres.PostgresContainer, string, error) {
	container, err := tcpostgres.Run(
		ctx,
		"postgres:18.3-alpine3.23",
		tcpostgres.WithDatabase("iam"),
		tcpostgres.WithUsername("iam"),
		tcpostgres.WithPassword("iam"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		return nil, "", err
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, "", err
	}

	return container, dsn, nil
}

func startRedis(ctx context.Context) (*tcredis.RedisContainer, string, error) {
	container, err := tcredis.Run(ctx, "redis:8.6.1-alpine3.23")
	if err != nil {
		return nil, "", err
	}

	addr, err := container.ConnectionString(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, "", err
	}

	// ConnectionString returns redis://host:port — strip the scheme
	const prefix = "redis://"
	if len(addr) > len(prefix) {
		addr = addr[len(prefix):]
	}

	return container, addr, nil
}

func applyMigrations(dsn string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer func() {
		_ = db.Close()
	}()

	if err = goose.SetDialect("postgres"); err != nil {
		return err
	}

	migrationsDir := findMigrationsDir()
	return goose.Up(db, migrationsDir)
}

// findMigrationsDir walks up from the current directory until it finds
// migrations/iam/, because the tests run from iam/tests/.
func findMigrationsDir() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	for i := 0; i < 5; i++ {
		candidate := filepath.Join(dir, "migrations", "iam")
		if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
			return candidate
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	panic("migrations/iam directory not found")
}

func TestRegisterLoginWhoamiLogout_HappyPath(t *testing.T) {
	ctx := context.Background()

	regResp, err := authClient.Register(ctx, &userv1.RegisterRequest{
		Info: &userv1.UserRegistrationInfo{
			Info:     &commonv1.UserInfo{Login: "alice"},
			Password: "password123",
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, regResp.GetUserUuid())

	loginResp, err := authSvc.Login(ctx, &authv1.LoginRequest{
		Login:    "alice",
		Password: "password123",
	})
	require.NoError(t, err)
	require.NotEmpty(t, loginResp.GetSessionUuid())

	sessionUUID := loginResp.GetSessionUuid()

	whoamiResp, err := authSvc.Whoami(ctx, &authv1.WhoamiRequest{SessionUuid: sessionUUID})
	require.NoError(t, err)
	require.Equal(t, regResp.GetUserUuid(), whoamiResp.GetUser().GetUuid())
	require.Equal(t, "alice", whoamiResp.GetUser().GetInfo().GetLogin())

	getUserResp, err := authClient.GetUser(ctx, &userv1.GetUserRequest{UserUuid: regResp.GetUserUuid()})
	require.NoError(t, err)
	require.Equal(t, "alice", getUserResp.GetUser().GetInfo().GetLogin())

	_, err = authSvc.Logout(ctx, &authv1.LogoutRequest{SessionUuid: sessionUUID})
	require.NoError(t, err)

	_, err = authSvc.Whoami(ctx, &authv1.WhoamiRequest{SessionUuid: sessionUUID})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestRegister_ValidationErrors(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		req  *userv1.RegisterRequest
		code codes.Code
	}{
		{
			name: "empty login",
			req: &userv1.RegisterRequest{
				Info: &userv1.UserRegistrationInfo{
					Info:     &commonv1.UserInfo{Login: ""},
					Password: "password123",
				},
			},
			code: codes.InvalidArgument,
		},
		{
			name: "weak password",
			req: &userv1.RegisterRequest{
				Info: &userv1.UserRegistrationInfo{
					Info:     &commonv1.UserInfo{Login: "carl"},
					Password: "short",
				},
			},
			code: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := authClient.Register(ctx, tt.req)
			requireGRPCCode(t, err, tt.code)
		})
	}
}

func TestRegister_AlreadyExists(t *testing.T) {
	ctx := context.Background()

	req := &userv1.RegisterRequest{
		Info: &userv1.UserRegistrationInfo{
			Info:     &commonv1.UserInfo{Login: "duplicate"},
			Password: "password123",
		},
	}

	_, err := authClient.Register(ctx, req)
	require.NoError(t, err)

	_, err = authClient.Register(ctx, req)
	requireGRPCCode(t, err, codes.AlreadyExists)
}

func TestLogin_InvalidCredentials(t *testing.T) {
	ctx := context.Background()

	_, err := authSvc.Login(ctx, &authv1.LoginRequest{Login: "ghost", Password: "any-password"})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestWhoami_EmptySession(t *testing.T) {
	ctx := context.Background()

	_, err := authSvc.Whoami(ctx, &authv1.WhoamiRequest{SessionUuid: ""})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// TestWhoami_NotFound covers the case of a session that does not exist in Redis
// (never existed, expired by TTL or was deleted). The code path is the same
// regardless of the reason: what matters is that Whoami returns Unauthenticated
// rather than Internal or OK.
func TestWhoami_NotFound(t *testing.T) {
	ctx := context.Background()

	_, err := authSvc.Whoami(ctx, &authv1.WhoamiRequest{
		SessionUuid: "11111111-2222-3333-4444-555555555555",
	})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

// TestWhoami_AfterLogout checks that after an explicit Logout the session is gone
// and Whoami returns Unauthenticated. The behaviour matches TTL expiry, but the
// trigger differs: Logout goes through IAMService.Logout → SessionRepository.Delete.
func TestWhoami_AfterLogout(t *testing.T) {
	ctx := context.Background()

	regResp, err := authClient.Register(ctx, &userv1.RegisterRequest{
		Info: &userv1.UserRegistrationInfo{
			Info:     &commonv1.UserInfo{Login: "logout-target"},
			Password: "password123",
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, regResp.GetUserUuid())

	loginResp, err := authSvc.Login(ctx, &authv1.LoginRequest{
		Login:    "logout-target",
		Password: "password123",
	})
	require.NoError(t, err)
	sessionUUID := loginResp.GetSessionUuid()
	require.NotEmpty(t, sessionUUID)

	// The session is valid right after Login
	_, err = authSvc.Whoami(ctx, &authv1.WhoamiRequest{SessionUuid: sessionUUID})
	require.NoError(t, err)

	// After Logout the session must be gone from Redis
	_, err = authSvc.Logout(ctx, &authv1.LogoutRequest{SessionUuid: sessionUUID})
	require.NoError(t, err)

	_, err = authSvc.Whoami(ctx, &authv1.WhoamiRequest{SessionUuid: sessionUUID})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestGetUser_InvalidUUID(t *testing.T) {
	ctx := context.Background()

	_, err := authClient.GetUser(ctx, &userv1.GetUserRequest{UserUuid: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestLogout_Idempotent(t *testing.T) {
	ctx := context.Background()

	// Logging out a non-existent session must not return an error
	_, err := authSvc.Logout(ctx, &authv1.LogoutRequest{SessionUuid: "11111111-2222-3333-4444-555555555555"})
	require.NoError(t, err)
}

func requireGRPCCode(t *testing.T, err error, expected codes.Code) {
	t.Helper()
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok, "expected a gRPC error, got: %v", err)
	require.Equal(t, expected, st.Code(), "message: %s", st.Message())
}
