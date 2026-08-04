//go:build apitest

package tests

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"

	iamApp "github.com/vixart/rocket-factory/iam/pkg/app"
	invApp "github.com/vixart/rocket-factory/inventory/pkg/app"
	"github.com/vixart/rocket-factory/order/internal/interceptor"
	"github.com/vixart/rocket-factory/order/pkg/app"
	"github.com/vixart/rocket-factory/order/tests/testutil"
	payApp "github.com/vixart/rocket-factory/payment/pkg/app"
	authv1 "github.com/vixart/rocket-factory/shared/pkg/proto/auth/v1"
	commonv1 "github.com/vixart/rocket-factory/shared/pkg/proto/common/v1"
	inventoryv1 "github.com/vixart/rocket-factory/shared/pkg/proto/inventory/v1"
	paymentv1 "github.com/vixart/rocket-factory/shared/pkg/proto/payment/v1"
	userv1 "github.com/vixart/rocket-factory/shared/pkg/proto/user/v1"
)

// Seeded part UUIDs and prices (from migrations/inventory/00002_seed_parts.sql)
const (
	HullAluminumUUID   = "550e8400-e29b-41d4-a716-446655440001" // 500000 kopecks (5000 RUB)
	HullTitaniumUUID   = "550e8400-e29b-41d4-a716-446655440002" // 1500000 kopecks (15000 RUB)
	EngineIonCUUID     = "550e8400-e29b-41d4-a716-446655440003" // 300000 kopecks (3000 RUB)
	EngineIonBUUID     = "550e8400-e29b-41d4-a716-446655440004" // 800000 kopecks (8000 RUB)
	ShieldEnergyUUID   = "550e8400-e29b-41d4-a716-446655440005" // 400000 kopecks (4000 RUB)
	WeaponLaserUUID    = "550e8400-e29b-41d4-a716-446655440006" // 250000 kopecks (2500 RUB)
	HullOutOfStockUUID = "550e8400-e29b-41d4-a716-446655440007" // 2000000 kopecks (20000 RUB), stock=0

	// Prices in kopecks
	HullAluminumPrice   = 500000
	HullTitaniumPrice   = 1500000
	EngineIonCPrice     = 300000
	EngineIonBPrice     = 800000
	ShieldEnergyPrice   = 400000
	WeaponLaserPrice    = 250000
	HullOutOfStockPrice = 2000000
)

const bufSize = 1024 * 1024

var (
	invLis *bufconn.Listener
	payLis *bufconn.Listener
	iamLis *bufconn.Listener

	inventoryClient inventoryv1.InventoryServiceClient
	paymentClient   paymentv1.PaymentServiceClient
	userClient      userv1.UserServiceClient
	authSvcClient   authv1.AuthServiceClient
	httpClient      = &http.Client{Timeout: 10 * time.Second}
	ts              *httptest.Server

	// orderDBPool is needed by tests that update the order status bypassing the API —
	// for example to test Cancel on an ASSEMBLED order (the API cannot reach that status
	// without the Kafka chain, which API tests do not run)
	orderDBPool *pgxpool.Pool

	// inventoryDBPool is needed by concurrency tests that prepare a part with a specific
	// stock_quantity directly in the database
	inventoryDBPool *pgxpool.Pool

	// defaultSessionUUID is the session of the default test user, registered in TestMain.
	// Every HTTP and gRPC helper uses it by default.
	defaultSessionUUID string

	// defaultUserUUID is the UUID of the default user, needed by tests that check the
	// link between an order and its owner.
	defaultUserUUID string
)

func invBufDialer(context.Context, string) (net.Conn, error) {
	return invLis.Dial()
}

func payBufDialer(context.Context, string) (net.Conn, error) {
	return payLis.Dial()
}

func iamBufDialer(context.Context, string) (net.Conn, error) {
	return iamLis.Dial()
}

// orderBaseURL returns the base URL for order HTTP tests
func orderBaseURL() string {
	return ts.URL
}

// startPostgres starts a PostgreSQL container and returns the connection DSN
func startPostgres(ctx context.Context, dbName, user, password string) (*tcpostgres.PostgresContainer, string, error) {
	container, err := tcpostgres.Run(
		ctx,
		"postgres:18.3-alpine3.23",
		tcpostgres.WithDatabase(dbName),
		tcpostgres.WithUsername(user),
		tcpostgres.WithPassword(password),
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
		return nil, "", err
	}

	return container, dsn, nil
}

// runMigrations applies the goose migrations from the given directory
func runMigrations(dsn, migrationsDir string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	absDir, err := filepath.Abs(migrationsDir)
	if err != nil {
		return err
	}

	return goose.Up(db, absDir)
}

// startRedis starts a Redis container for IAM sessions and returns host:port without
// the "redis://" prefix, so it can go straight into redis.Options{Addr: ...}
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

	const prefix = "redis://"
	if len(addr) > len(prefix) {
		addr = addr[len(prefix):]
	}

	return container, addr, nil
}

// TestMain starts every service before the tests and stops them afterwards.
// Since week 6 the environment also includes IAM (PostgreSQL + Redis + gRPC server):
// it is needed both for the authentication tests and to register the default session
// every ordinary Order/Inventory test runs under.
func TestMain(m *testing.M) {
	ctx := context.Background()

	// 1. Start the PostgreSQL container for the order service
	orderContainer, orderDSN, err := startPostgres(
		ctx,
		"order-service",
		"order-service-user",
		"order-service-password",
	)
	if err != nil {
		panic(err)
	}

	// 2. Start the PostgreSQL container for the inventory service
	inventoryContainer, inventoryDSN, err := startPostgres(
		ctx,
		"inventory-service",
		"inventory-service-user",
		"inventory-service-password",
	)
	if err != nil {
		panic(err)
	}

	// 3. Start the PostgreSQL container for IAM
	iamContainer, iamDSN, err := startPostgres(
		ctx,
		"iam-service",
		"iam-service-user",
		"iam-service-password",
	)
	if err != nil {
		panic(err)
	}

	// 4. Start Redis for IAM sessions
	redisContainer, redisAddr, err := startRedis(ctx)
	if err != nil {
		panic(err)
	}

	// 5. Apply the migrations
	if err = runMigrations(orderDSN, "../../migrations/order"); err != nil {
		panic(err)
	}
	if err = runMigrations(inventoryDSN, "../../migrations/inventory"); err != nil {
		panic(err)
	}
	if err = runMigrations(iamDSN, "../../migrations/iam"); err != nil {
		panic(err)
	}

	// 6. Create the pgxpools
	orderPool, err := pgxpool.New(ctx, orderDSN)
	if err != nil {
		panic(err)
	}
	orderDBPool = orderPool

	inventoryPool, err := pgxpool.New(ctx, inventoryDSN)
	if err != nil {
		panic(err)
	}
	inventoryDBPool = inventoryPool

	iamPool, err := pgxpool.New(ctx, iamDSN)
	if err != nil {
		panic(err)
	}

	// 7. Redis client for IAM
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})

	// 8. TxManager for the order service
	txManager, err := manager.New(trmpgx.NewDefaultFactory(orderPool))
	if err != nil {
		panic(err)
	}

	// 9. IAM gRPC over bufconn (needed by the Inventory server for its auth interceptor
	//    and by the Order HTTP handler for its middleware)
	iamLis = bufconn.Listen(bufSize)
	iamGRPCServer := iamApp.NewGRPCServer(iamPool, rdb, time.Hour, bcrypt.MinCost)
	go func() {
		if iamServeErr := iamGRPCServer.Serve(iamLis); iamServeErr != nil {
			panic(iamServeErr)
		}
	}()

	iamConn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(iamBufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(err)
	}
	userClient = userv1.NewUserServiceClient(iamConn)
	authSvcClient = authv1.NewAuthServiceClient(iamConn)

	// 10. Inventory gRPC over bufconn — now with the auth interceptor
	invLis = bufconn.Listen(bufSize)
	invGRPCServer := grpc.NewServer(invApp.Interceptors(authSvcClient)...)
	invTxManager, err := manager.New(trmpgx.NewDefaultFactory(inventoryPool))
	invApp.RegisterServices(invTxManager, invGRPCServer, inventoryPool)
	go func() {
		if invServeErr := invGRPCServer.Serve(invLis); invServeErr != nil {
			panic(invServeErr)
		}
	}()

	invConn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(invBufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(interceptor.SessionForwarder()),
	)
	if err != nil {
		panic(err)
	}
	inventoryClient = inventoryv1.NewInventoryServiceClient(invConn)

	// 11. Payment gRPC over bufconn (no auth)
	payLis = bufconn.Listen(bufSize)
	payGRPCServer := grpc.NewServer(payApp.Interceptors()...)
	payApp.RegisterServices(payGRPCServer)
	go func() {
		if payServeErr := payGRPCServer.Serve(payLis); payServeErr != nil {
			panic(payServeErr)
		}
	}()

	payConn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(payBufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(err)
	}
	paymentClient = paymentv1.NewPaymentServiceClient(payConn)

	// 12. Order HTTP over httptest, with a real IAM client for the middleware
	orderServer, err := app.NewHTTPHandler(orderPool, txManager, inventoryClient, paymentClient, authSvcClient)
	if err != nil {
		panic(err)
	}
	ts = httptest.NewServer(orderServer)

	// 13. Register the default user and log in: every ordinary test runs under this
	//     session, and defaultSessionUUID / defaultUserUUID become the global context.
	//
	defaultSessionUUID, defaultUserUUID, err = registerAndLoginCtx(ctx, "default-user", "password123")
	if err != nil {
		panic(err)
	}

	code := m.Run()

	ts.Close()
	if err = invConn.Close(); err != nil {
		panic(err)
	}
	if err = payConn.Close(); err != nil {
		panic(err)
	}
	if err = iamConn.Close(); err != nil {
		panic(err)
	}
	invGRPCServer.Stop()
	payGRPCServer.Stop()
	iamGRPCServer.Stop()

	orderPool.Close()
	inventoryPool.Close()
	iamPool.Close()
	_ = rdb.Close()

	if err = orderContainer.Terminate(ctx); err != nil {
		panic(err)
	}
	if err = inventoryContainer.Terminate(ctx); err != nil {
		panic(err)
	}
	if err = iamContainer.Terminate(ctx); err != nil {
		panic(err)
	}
	if err = redisContainer.Terminate(ctx); err != nil {
		panic(err)
	}

	os.Exit(code)
}

// authCtx adds session-uuid to the outgoing metadata, required by direct gRPC calls to
// InventoryService, whose server demands metadata.session-uuid (auth interceptor).
func authCtx(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "session-uuid", defaultSessionUUID)
}

// authCtxWith is authCtx with an arbitrary session, used by tests that need to check
// interceptor behaviour for a specific value.
func authCtxWith(ctx context.Context, sessionUUID string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "session-uuid", sessionUUID)
}

// registerAndLogin registers a new user and logs in, returning the session UUID and the
// user UUID. It serves tests that need their own session, different from the default
// one (checking user_uuid taken from the session, for example).
func registerAndLogin(t *testing.T, login, password string) (sessionUUID, userUUID string) {
	t.Helper()
	sUUID, uUUID, err := registerAndLoginCtx(context.Background(), login, password)
	require.NoError(t, err)
	return sUUID, uUUID
}

// registerAndLoginCtx is registerAndLogin without a *testing.T dependency, needed in
// TestMain to register the default user before the first test runs.
func registerAndLoginCtx(ctx context.Context, login, password string) (sessionUUID, userUUID string, err error) {
	regResp, err := userClient.Register(ctx, &userv1.RegisterRequest{
		Info: &userv1.UserRegistrationInfo{
			Info:     &commonv1.UserInfo{Login: login},
			Password: password,
		},
	})
	if err != nil {
		return "", "", err
	}

	loginResp, err := authSvcClient.Login(ctx, &authv1.LoginRequest{
		Login:    login,
		Password: password,
	})
	if err != nil {
		return "", "", err
	}

	return loginResp.GetSessionUuid(), regResp.GetUserUuid(), nil
}

// HTTP request and response types

// CreateOrderRequest is the request body for creating an order.
// Since week 6 the user_uuid field is gone from the HTTP body (it comes from the
// authenticated session). UserUUID stays in the struct for compatibility with existing
// tests but is not serialized to JSON (the "-" tag); the actual owner is decided by the
// default defaultSessionUUID session or one created explicitly via registerAndLogin.
type CreateOrderRequest struct {
	UserUUID   string  `json:"-"`
	HullUUID   string  `json:"hull_uuid"`
	EngineUUID string  `json:"engine_uuid"`
	ShieldUUID *string `json:"shield_uuid,omitempty"`
	WeaponUUID *string `json:"weapon_uuid,omitempty"`
}

// CreateOrderResponse is the response to an order creation
type CreateOrderResponse struct {
	OrderUUID  string `json:"order_uuid"`
	TotalPrice int64  `json:"total_price"`
}

// PayOrderRequest is the request body for paying for an order
type PayOrderRequest struct {
	PaymentMethod string `json:"payment_method"`
}

// PayOrderResponse is the response to an order payment
type PayOrderResponse struct {
	TransactionUUID string `json:"transaction_uuid"`
}

// CancelOrderResponse is the response to an order cancellation (empty)
type CancelOrderResponse struct{}

// OrderDTO is an order as returned by the API
type OrderDTO struct {
	OrderUUID       string  `json:"order_uuid"`
	UserUUID        string  `json:"user_uuid"`
	HullUUID        string  `json:"hull_uuid"`
	EngineUUID      string  `json:"engine_uuid"`
	ShieldUUID      *string `json:"shield_uuid"`
	WeaponUUID      *string `json:"weapon_uuid"`
	TotalPrice      int64   `json:"total_price"`
	TransactionUUID *string `json:"transaction_uuid"`
	PaymentMethod   *string `json:"payment_method"`
	Status          string  `json:"status"`
	CreatedAt       string  `json:"created_at"`
}

// ErrorResponse is an API error response
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// HTTP helper functions

// withDefaultAuth sets the Authorization header of the default session.
// Every HTTP helper below uses it: since week 6 the middleware answers 401 without a
// Bearer session.
func withDefaultAuth(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+defaultSessionUUID)
}

func createOrder(t *testing.T, req *CreateOrderRequest) (*CreateOrderResponse, *http.Response) {
	t.Helper()

	jsonBody, err := json.Marshal(req)
	require.NoError(t, err)

	httpReq, err := http.NewRequest(http.MethodPost, orderBaseURL()+"/api/v1/orders", bytes.NewReader(jsonBody))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")
	withDefaultAuth(httpReq)

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)

	if resp.StatusCode == http.StatusCreated {
		var result CreateOrderResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		return &result, resp
	}

	return nil, resp
}

func getOrder(t *testing.T, orderUUID string) (*OrderDTO, *http.Response) {
	t.Helper()

	httpReq, err := http.NewRequest(http.MethodGet, orderBaseURL()+"/api/v1/orders/"+orderUUID, nil)
	require.NoError(t, err)
	withDefaultAuth(httpReq)

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)

	if resp.StatusCode == http.StatusOK {
		var result OrderDTO
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		return &result, resp
	}

	return nil, resp
}

func payOrder(t *testing.T, orderUUID string, req *PayOrderRequest) (*PayOrderResponse, *http.Response) {
	t.Helper()

	jsonBody, err := json.Marshal(req)
	require.NoError(t, err)

	httpReq, err := http.NewRequest(http.MethodPost, orderBaseURL()+"/api/v1/orders/"+orderUUID+"/pay", bytes.NewReader(jsonBody))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")
	withDefaultAuth(httpReq)

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)

	if resp.StatusCode == http.StatusOK {
		var result PayOrderResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		return &result, resp
	}

	return nil, resp
}

func cancelOrder(t *testing.T, orderUUID string) (*CancelOrderResponse, *http.Response) {
	t.Helper()

	httpReq, err := http.NewRequest(http.MethodPost, orderBaseURL()+"/api/v1/orders/"+orderUUID+"/cancel", nil)
	require.NoError(t, err)
	withDefaultAuth(httpReq)

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)

	if resp.StatusCode == http.StatusOK {
		var result CancelOrderResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		return &result, resp
	}

	return nil, resp
}

// InventoryService tests (gRPC)

func TestInventory_GetPart_Success(t *testing.T) {
	resp, err := inventoryClient.GetPart(authCtx(context.Background()), &inventoryv1.GetPartRequest{
		Uuid: HullAluminumUUID,
	})
	require.NoError(t, err)

	part := resp.GetPart()
	assert.Equal(t, HullAluminumUUID, part.GetUuid())
	assert.Equal(t, int64(HullAluminumPrice), part.GetPrice())
	assert.Equal(t, inventoryv1.PartType_PART_TYPE_HULL, part.GetPartType())
	assert.NotEmpty(t, part.GetName())
	assert.NotEmpty(t, part.GetDescription(), "description must not be empty")
	assert.NotNil(t, part.GetCreatedAt())
}

func TestInventory_GetPart_AllTypes(t *testing.T) {
	testCases := []struct {
		name        string
		uuid        string
		price       int64
		partType    inventoryv1.PartType
		description string
	}{
		{"Hull Aluminum", HullAluminumUUID, HullAluminumPrice, inventoryv1.PartType_PART_TYPE_HULL, "Lightweight hull for small ships"},
		{"Hull Titanium", HullTitaniumUUID, HullTitaniumPrice, inventoryv1.PartType_PART_TYPE_HULL, "Durable hull for medium ships"},
		{"Engine Ion C", EngineIonCUUID, EngineIonCPrice, inventoryv1.PartType_PART_TYPE_ENGINE, "Basic class C ion engine"},
		{"Engine Ion B", EngineIonBUUID, EngineIonBPrice, inventoryv1.PartType_PART_TYPE_ENGINE, "Improved class B ion engine"},
		{"Shield Energy", ShieldEnergyUUID, ShieldEnergyPrice, inventoryv1.PartType_PART_TYPE_SHIELD, "Standard energy shield"},
		{"Weapon Laser", WeaponLaserUUID, WeaponLaserPrice, inventoryv1.PartType_PART_TYPE_WEAPON, "Precise laser cannon"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := inventoryClient.GetPart(authCtx(context.Background()), &inventoryv1.GetPartRequest{
				Uuid: tc.uuid,
			})
			require.NoError(t, err)

			part := resp.GetPart()
			assert.Equal(t, tc.uuid, part.GetUuid())
			assert.Equal(t, tc.price, part.GetPrice())
			assert.Equal(t, tc.partType, part.GetPartType())
			assert.Equal(t, tc.description, part.GetDescription())
		})
	}
}

func TestInventory_GetPart_NotFound(t *testing.T) {
	_, err := inventoryClient.GetPart(authCtx(context.Background()), &inventoryv1.GetPartRequest{
		Uuid: uuid.New().String(),
	})
	require.Error(t, err)
	testutil.AssertGRPCStatus(t, err, codes.NotFound)
}

func TestInventory_GetPart_EmptyUUID(t *testing.T) {
	_, err := inventoryClient.GetPart(authCtx(context.Background()), &inventoryv1.GetPartRequest{
		Uuid: "",
	})
	require.Error(t, err)
	testutil.AssertGRPCStatus(t, err, codes.InvalidArgument)
}

func TestInventory_GetPart_InvalidUUID(t *testing.T) {
	_, err := inventoryClient.GetPart(authCtx(context.Background()), &inventoryv1.GetPartRequest{
		Uuid: "invalid-uuid-format",
	})
	require.Error(t, err)
	testutil.AssertGRPCStatus(t, err, codes.InvalidArgument)
}

func TestInventory_ListParts_All(t *testing.T) {
	resp, err := inventoryClient.ListParts(authCtx(context.Background()), &inventoryv1.ListPartsRequest{
		PartType: inventoryv1.PartType_PART_TYPE_UNSPECIFIED,
	})
	require.NoError(t, err)
	assert.Len(t, resp.GetParts(), 7)
}

func TestInventory_ListParts_ByType_Hull(t *testing.T) {
	resp, err := inventoryClient.ListParts(authCtx(context.Background()), &inventoryv1.ListPartsRequest{
		PartType: inventoryv1.PartType_PART_TYPE_HULL,
	})
	require.NoError(t, err)
	assert.Len(t, resp.GetParts(), 3) // aluminium, titanium, plasma (stock=0)

	for _, part := range resp.GetParts() {
		assert.Equal(t, inventoryv1.PartType_PART_TYPE_HULL, part.GetPartType())
	}
}

func TestInventory_ListParts_ByType_Engine(t *testing.T) {
	resp, err := inventoryClient.ListParts(authCtx(context.Background()), &inventoryv1.ListPartsRequest{
		PartType: inventoryv1.PartType_PART_TYPE_ENGINE,
	})
	require.NoError(t, err)
	assert.Len(t, resp.GetParts(), 2)

	for _, part := range resp.GetParts() {
		assert.Equal(t, inventoryv1.PartType_PART_TYPE_ENGINE, part.GetPartType())
	}
}

func TestInventory_ListParts_ByType_Shield(t *testing.T) {
	resp, err := inventoryClient.ListParts(authCtx(context.Background()), &inventoryv1.ListPartsRequest{
		PartType: inventoryv1.PartType_PART_TYPE_SHIELD,
	})
	require.NoError(t, err)
	assert.Len(t, resp.GetParts(), 1)
	assert.Equal(t, ShieldEnergyUUID, resp.GetParts()[0].GetUuid())
}

func TestInventory_ListParts_ByType_Weapon(t *testing.T) {
	resp, err := inventoryClient.ListParts(authCtx(context.Background()), &inventoryv1.ListPartsRequest{
		PartType: inventoryv1.PartType_PART_TYPE_WEAPON,
	})
	require.NoError(t, err)
	assert.Len(t, resp.GetParts(), 1)
	assert.Equal(t, WeaponLaserUUID, resp.GetParts()[0].GetUuid())
}

func TestInventory_ListParts_SortedByName(t *testing.T) {
	resp, err := inventoryClient.ListParts(authCtx(context.Background()), &inventoryv1.ListPartsRequest{
		PartType: inventoryv1.PartType_PART_TYPE_UNSPECIFIED,
	})
	require.NoError(t, err)

	parts := resp.GetParts()
	for i := 1; i < len(parts); i++ {
		assert.LessOrEqual(t, parts[i-1].GetName(), parts[i].GetName(),
			"parts must be sorted by name in alphabetical order")
	}
}

// ListParts.uuids tests

func TestInventory_ListParts_ByUuids_Success(t *testing.T) {
	uuids := []string{HullAluminumUUID, EngineIonCUUID, ShieldEnergyUUID}

	resp, err := inventoryClient.ListParts(authCtx(context.Background()), &inventoryv1.ListPartsRequest{
		Uuids: uuids,
	})
	require.NoError(t, err)
	assert.Len(t, resp.GetParts(), 3)

	// Check that the expected parts came back
	returnedUUIDs := make([]string, len(resp.GetParts()))
	for i, part := range resp.GetParts() {
		returnedUUIDs[i] = part.GetUuid()
	}
	assert.ElementsMatch(t, uuids, returnedUUIDs)
}

func TestInventory_ListParts_ByUuids_PreservesOrder(t *testing.T) {
	// Request in a specific order: Engine, Hull, Weapon
	uuids := []string{EngineIonCUUID, HullAluminumUUID, WeaponLaserUUID}

	resp, err := inventoryClient.ListParts(authCtx(context.Background()), &inventoryv1.ListPartsRequest{
		Uuids: uuids,
	})
	require.NoError(t, err)
	assert.Len(t, resp.GetParts(), 3)

	// Check that the order matches the request
	for i, part := range resp.GetParts() {
		assert.Equal(t, uuids[i], part.GetUuid(),
			"part at index %d must follow the order of the requested UUIDs", i)
	}
}

func TestInventory_ListParts_ByUuids_IgnoresPartType(t *testing.T) {
	// A request with uuids AND part_type — part_type must be ignored
	uuids := []string{HullAluminumUUID, EngineIonCUUID}

	resp, err := inventoryClient.ListParts(authCtx(context.Background()), &inventoryv1.ListPartsRequest{
		Uuids:    uuids,
		PartType: inventoryv1.PartType_PART_TYPE_WEAPON, // must be ignored
	})
	require.NoError(t, err)
	assert.Len(t, resp.GetParts(), 2)

	// Check that we got the hull and the engine, not weapons
	assert.Equal(t, HullAluminumUUID, resp.GetParts()[0].GetUuid())
	assert.Equal(t, EngineIonCUUID, resp.GetParts()[1].GetUuid())
}

func TestInventory_ListParts_ByUuids_NotFound(t *testing.T) {
	// Include one non-existent UUID
	nonExistentUUID := uuid.New().String()
	uuids := []string{HullAluminumUUID, nonExistentUUID, EngineIonCUUID}

	_, err := inventoryClient.ListParts(authCtx(context.Background()), &inventoryv1.ListPartsRequest{
		Uuids: uuids,
	})
	require.Error(t, err)
	testutil.AssertGRPCStatus(t, err, codes.NotFound)
}

func TestInventory_ListParts_ByUuids_InvalidUUID(t *testing.T) {
	uuids := []string{HullAluminumUUID, "invalid-uuid-format"}

	_, err := inventoryClient.ListParts(authCtx(context.Background()), &inventoryv1.ListPartsRequest{
		Uuids: uuids,
	})
	require.Error(t, err)
	testutil.AssertGRPCStatus(t, err, codes.InvalidArgument)
}

func TestInventory_ListParts_ByUuids_SingleUUID(t *testing.T) {
	uuids := []string{WeaponLaserUUID}

	resp, err := inventoryClient.ListParts(authCtx(context.Background()), &inventoryv1.ListPartsRequest{
		Uuids: uuids,
	})
	require.NoError(t, err)
	assert.Len(t, resp.GetParts(), 1)
	assert.Equal(t, WeaponLaserUUID, resp.GetParts()[0].GetUuid())
	assert.Equal(t, int64(WeaponLaserPrice), resp.GetParts()[0].GetPrice())
}

func TestInventory_ListParts_ByUuids_AllParts(t *testing.T) {
	// Request all six parts by UUID
	uuids := []string{
		HullAluminumUUID, HullTitaniumUUID,
		EngineIonCUUID, EngineIonBUUID,
		ShieldEnergyUUID, WeaponLaserUUID,
	}

	resp, err := inventoryClient.ListParts(authCtx(context.Background()), &inventoryv1.ListPartsRequest{
		Uuids: uuids,
	})
	require.NoError(t, err)
	assert.Len(t, resp.GetParts(), 6)

	// Check that the order matches the request order
	for i, part := range resp.GetParts() {
		assert.Equal(t, uuids[i], part.GetUuid())
	}
}

func TestInventory_ListParts_ByUuids_EmptyList(t *testing.T) {
	// An empty UUID list must return every part (type filter is UNSPECIFIED)
	resp, err := inventoryClient.ListParts(authCtx(context.Background()), &inventoryv1.ListPartsRequest{
		Uuids: []string{},
	})
	require.NoError(t, err)
	assert.Len(t, resp.GetParts(), 7)
}

// PaymentService tests (gRPC)

func TestPayment_PayOrder_Success_Card(t *testing.T) {
	resp, err := paymentClient.PayOrder(context.Background(), &paymentv1.PayOrderRequest{
		OrderUuid:     uuid.New().String(),
		PaymentMethod: paymentv1.PaymentMethod_PAYMENT_METHOD_CARD,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.GetTransactionUuid())

	// Check that the transaction UUID is valid
	_, err = uuid.Parse(resp.GetTransactionUuid())
	assert.NoError(t, err)
}

func TestPayment_PayOrder_Success_SBP(t *testing.T) {
	resp, err := paymentClient.PayOrder(context.Background(), &paymentv1.PayOrderRequest{
		OrderUuid:     uuid.New().String(),
		PaymentMethod: paymentv1.PaymentMethod_PAYMENT_METHOD_SBP,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.GetTransactionUuid())
}

func TestPayment_PayOrder_Success_CreditCard(t *testing.T) {
	resp, err := paymentClient.PayOrder(context.Background(), &paymentv1.PayOrderRequest{
		OrderUuid:     uuid.New().String(),
		PaymentMethod: paymentv1.PaymentMethod_PAYMENT_METHOD_CREDIT_CARD,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.GetTransactionUuid())
}

func TestPayment_PayOrder_Success_InvestorMoney(t *testing.T) {
	resp, err := paymentClient.PayOrder(context.Background(), &paymentv1.PayOrderRequest{
		OrderUuid:     uuid.New().String(),
		PaymentMethod: paymentv1.PaymentMethod_PAYMENT_METHOD_INVESTOR_MONEY,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.GetTransactionUuid())
}

func TestPayment_PayOrder_EmptyOrderUUID(t *testing.T) {
	_, err := paymentClient.PayOrder(context.Background(), &paymentv1.PayOrderRequest{
		OrderUuid:     "",
		PaymentMethod: paymentv1.PaymentMethod_PAYMENT_METHOD_CARD,
	})
	require.Error(t, err)
	testutil.AssertGRPCStatus(t, err, codes.InvalidArgument)
}

func TestPayment_PayOrder_UnspecifiedMethod(t *testing.T) {
	_, err := paymentClient.PayOrder(context.Background(), &paymentv1.PayOrderRequest{
		OrderUuid:     uuid.New().String(),
		PaymentMethod: paymentv1.PaymentMethod_PAYMENT_METHOD_UNSPECIFIED,
	})
	require.Error(t, err)
	testutil.AssertGRPCStatus(t, err, codes.InvalidArgument)
}

func TestPayment_PayOrder_UniqueTransactions(t *testing.T) {
	orderUUID := uuid.New().String()

	resp1, err := paymentClient.PayOrder(context.Background(), &paymentv1.PayOrderRequest{
		OrderUuid:     orderUUID,
		PaymentMethod: paymentv1.PaymentMethod_PAYMENT_METHOD_CARD,
	})
	require.NoError(t, err)

	resp2, err := paymentClient.PayOrder(context.Background(), &paymentv1.PayOrderRequest{
		OrderUuid:     orderUUID,
		PaymentMethod: paymentv1.PaymentMethod_PAYMENT_METHOD_CARD,
	})
	require.NoError(t, err)

	assert.NotEqual(t, resp1.GetTransactionUuid(), resp2.GetTransactionUuid(),
		"every payment must generate a unique transaction UUID")
}

// OrderService tests (HTTP)

func TestOrder_Create_Success_MinimalParts(t *testing.T) {
	req := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullAluminumUUID,
		EngineUUID: EngineIonCUUID,
	}

	result, resp := createOrder(t, req)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.OrderUUID)
	assert.Equal(t, int64(HullAluminumPrice+EngineIonCPrice), result.TotalPrice)
}

func TestOrder_Create_Success_AllParts(t *testing.T) {
	req := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullTitaniumUUID,
		EngineUUID: EngineIonBUUID,
		ShieldUUID: new(ShieldEnergyUUID),
		WeaponUUID: new(WeaponLaserUUID),
	}

	result, resp := createOrder(t, req)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.OrderUUID)

	expectedTotal := int64(HullTitaniumPrice + EngineIonBPrice + ShieldEnergyPrice + WeaponLaserPrice)
	assert.Equal(t, expectedTotal, result.TotalPrice)
}

func TestOrder_Create_VerifyTotalPrice(t *testing.T) {
	req := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullAluminumUUID, // 500000
		EngineUUID: EngineIonCUUID,   // 300000
	}

	result, resp := createOrder(t, req)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, int64(800000), result.TotalPrice, "500000 + 300000 = 800000")
}

func TestOrder_Create_HullNotFound(t *testing.T) {
	req := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   uuid.New().String(),
		EngineUUID: EngineIonCUUID,
	}

	_, resp := createOrder(t, req)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestOrder_Create_EngineNotFound(t *testing.T) {
	req := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullAluminumUUID,
		EngineUUID: uuid.New().String(),
	}

	_, resp := createOrder(t, req)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestOrder_Create_ShieldNotFound(t *testing.T) {
	req := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullAluminumUUID,
		EngineUUID: EngineIonCUUID,
		ShieldUUID: new(uuid.New().String()),
	}

	_, resp := createOrder(t, req)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestOrder_Create_WeaponNotFound(t *testing.T) {
	req := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullAluminumUUID,
		EngineUUID: EngineIonCUUID,
		WeaponUUID: new(uuid.New().String()),
	}

	_, resp := createOrder(t, req)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestOrder_Get_Success(t *testing.T) {
	// Create an order first
	createReq := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullAluminumUUID,
		EngineUUID: EngineIonCUUID,
	}
	createResult, createResp := createOrder(t, createReq)
	_ = createResp.Body.Close()
	require.NotNil(t, createResult)

	// Fetch the order
	order, resp := getOrder(t, createResult.OrderUUID)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NotNil(t, order)
	assert.Equal(t, createResult.OrderUUID, order.OrderUUID)
	assert.Equal(t, HullAluminumUUID, order.HullUUID)
	assert.Equal(t, EngineIonCUUID, order.EngineUUID)
	assert.Equal(t, createResult.TotalPrice, order.TotalPrice)
}

func TestOrder_Get_VerifyStatus_PendingPayment(t *testing.T) {
	// Create an order
	createReq := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullAluminumUUID,
		EngineUUID: EngineIonCUUID,
	}
	createResult, createResp := createOrder(t, createReq)
	_ = createResp.Body.Close()
	require.NotNil(t, createResult)

	// Fetch it and check the status
	order, resp := getOrder(t, createResult.OrderUUID)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "PENDING_PAYMENT", order.Status)
}

func TestOrder_Get_NotFound(t *testing.T) {
	_, resp := getOrder(t, uuid.New().String())
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestOrder_Pay_Success_Card(t *testing.T) {
	// Create an order
	createReq := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullAluminumUUID,
		EngineUUID: EngineIonCUUID,
	}
	createResult, createResp := createOrder(t, createReq)
	_ = createResp.Body.Close()
	require.NotNil(t, createResult)

	// Pay for the order
	payReq := &PayOrderRequest{PaymentMethod: "CARD"}
	payResult, resp := payOrder(t, createResult.OrderUUID, payReq)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NotNil(t, payResult)
	assert.NotEmpty(t, payResult.TransactionUUID)
}

func TestOrder_Pay_VerifyStatusChange(t *testing.T) {
	// Create an order
	createReq := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullAluminumUUID,
		EngineUUID: EngineIonCUUID,
	}
	createResult, createResp := createOrder(t, createReq)
	_ = createResp.Body.Close()
	require.NotNil(t, createResult)

	// Pay for the order
	payReq := &PayOrderRequest{PaymentMethod: "CARD"}
	_, payResp := payOrder(t, createResult.OrderUUID, payReq)
	_ = payResp.Body.Close()

	// Fetch it and check the status changed to PAID
	order, getResp := getOrder(t, createResult.OrderUUID)
	defer func() { _ = getResp.Body.Close() }()

	require.Equal(t, http.StatusOK, getResp.StatusCode)
	assert.Equal(t, "PAID", order.Status)
	assert.NotNil(t, order.TransactionUUID)
	assert.NotNil(t, order.PaymentMethod)
	assert.Equal(t, "CARD", *order.PaymentMethod)
}

func TestOrder_Pay_NotFound(t *testing.T) {
	payReq := &PayOrderRequest{PaymentMethod: "CARD"}
	_, resp := payOrder(t, uuid.New().String(), payReq)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestOrder_Pay_AlreadyPaid(t *testing.T) {
	// Create an order
	createReq := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullAluminumUUID,
		EngineUUID: EngineIonCUUID,
	}
	createResult, createResp := createOrder(t, createReq)
	_ = createResp.Body.Close()
	require.NotNil(t, createResult)

	// Pay for the order the first time
	payReq := &PayOrderRequest{PaymentMethod: "CARD"}
	_, payResp1 := payOrder(t, createResult.OrderUUID, payReq)
	_ = payResp1.Body.Close()

	// Try to pay again — a conflict error is expected
	_, payResp2 := payOrder(t, createResult.OrderUUID, payReq)
	defer func() { _ = payResp2.Body.Close() }()

	require.Equal(t, http.StatusConflict, payResp2.StatusCode)
}

func TestOrder_Pay_AlreadyCancelled(t *testing.T) {
	// Create an order
	createReq := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullAluminumUUID,
		EngineUUID: EngineIonCUUID,
	}
	createResult, createResp := createOrder(t, createReq)
	_ = createResp.Body.Close()
	require.NotNil(t, createResult)

	// Cancel the order
	_, cancelResp := cancelOrder(t, createResult.OrderUUID)
	_ = cancelResp.Body.Close()

	// Try to pay for a cancelled order — a conflict error is expected
	payReq := &PayOrderRequest{PaymentMethod: "CARD"}
	_, payResp := payOrder(t, createResult.OrderUUID, payReq)
	defer func() { _ = payResp.Body.Close() }()

	require.Equal(t, http.StatusConflict, payResp.StatusCode)
}

func TestOrder_Cancel_Success(t *testing.T) {
	// Create an order
	createReq := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullAluminumUUID,
		EngineUUID: EngineIonCUUID,
	}
	createResult, createResp := createOrder(t, createReq)
	_ = createResp.Body.Close()
	require.NotNil(t, createResult)

	// Cancel the order
	_, resp := cancelOrder(t, createResult.OrderUUID)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestOrder_Cancel_VerifyStatusChange(t *testing.T) {
	// Create an order
	createReq := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullAluminumUUID,
		EngineUUID: EngineIonCUUID,
	}
	createResult, createResp := createOrder(t, createReq)
	_ = createResp.Body.Close()
	require.NotNil(t, createResult)

	// Cancel the order
	_, cancelResp := cancelOrder(t, createResult.OrderUUID)
	_ = cancelResp.Body.Close()

	// Fetch it and check the status changed to CANCELLED
	order, getResp := getOrder(t, createResult.OrderUUID)
	defer func() { _ = getResp.Body.Close() }()

	require.Equal(t, http.StatusOK, getResp.StatusCode)
	assert.Equal(t, "CANCELLED", order.Status)
}

func TestOrder_Cancel_NotFound(t *testing.T) {
	_, resp := cancelOrder(t, uuid.New().String())
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestOrder_Cancel_AlreadyPaid(t *testing.T) {
	// Create an order
	createReq := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullAluminumUUID,
		EngineUUID: EngineIonCUUID,
	}
	createResult, createResp := createOrder(t, createReq)
	_ = createResp.Body.Close()
	require.NotNil(t, createResult)

	// Pay for the order
	payReq := &PayOrderRequest{PaymentMethod: "CARD"}
	_, payResp := payOrder(t, createResult.OrderUUID, payReq)
	_ = payResp.Body.Close()

	// Try to cancel a paid order — a conflict error is expected
	_, cancelResp := cancelOrder(t, createResult.OrderUUID)
	defer func() { _ = cancelResp.Body.Close() }()

	require.Equal(t, http.StatusConflict, cancelResp.StatusCode)
}

func TestOrder_Cancel_AlreadyCancelled(t *testing.T) {
	// Create an order
	createReq := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullAluminumUUID,
		EngineUUID: EngineIonCUUID,
	}
	createResult, createResp := createOrder(t, createReq)
	_ = createResp.Body.Close()
	require.NotNil(t, createResult)

	// Cancel the order first time
	_, cancelResp1 := cancelOrder(t, createResult.OrderUUID)
	_ = cancelResp1.Body.Close()

	// Try to cancel again — a conflict error is expected
	_, cancelResp2 := cancelOrder(t, createResult.OrderUUID)
	defer func() { _ = cancelResp2.Body.Close() }()

	require.Equal(t, http.StatusConflict, cancelResp2.StatusCode)
}

// Additional validation tests

func TestOrder_Create_WithWeaponOnly(t *testing.T) {
	req := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullAluminumUUID,
		EngineUUID: EngineIonCUUID,
		WeaponUUID: new(WeaponLaserUUID),
	}

	result, resp := createOrder(t, req)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.NotNil(t, result)
	expectedTotal := int64(HullAluminumPrice + EngineIonCPrice + WeaponLaserPrice)
	assert.Equal(t, expectedTotal, result.TotalPrice)
}

func TestOrder_Pay_AllMethods(t *testing.T) {
	methods := []string{"CARD", "SBP", "CREDIT_CARD", "INVESTOR_MONEY"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			// Create an order
			createReq := &CreateOrderRequest{
				UserUUID:   uuid.New().String(),
				HullUUID:   HullAluminumUUID,
				EngineUUID: EngineIonCUUID,
			}
			createResult, createResp := createOrder(t, createReq)
			_ = createResp.Body.Close()
			require.NotNil(t, createResult)

			// Pay with this method
			payReq := &PayOrderRequest{PaymentMethod: method}
			payResult, resp := payOrder(t, createResult.OrderUUID, payReq)
			defer func() { _ = resp.Body.Close() }()

			require.Equal(t, http.StatusOK, resp.StatusCode)
			require.NotNil(t, payResult)
			assert.NotEmpty(t, payResult.TransactionUUID)

			// Check that the payment method was persisted
			order, getResp := getOrder(t, createResult.OrderUUID)
			_ = getResp.Body.Close()
			require.NotNil(t, order.PaymentMethod)
			assert.Equal(t, method, *order.PaymentMethod)
		})
	}
}

func TestOrder_Get_WithOptionalParts(t *testing.T) {
	shieldUUID := ShieldEnergyUUID
	weaponUUID := WeaponLaserUUID
	req := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullAluminumUUID,
		EngineUUID: EngineIonCUUID,
		ShieldUUID: &shieldUUID,
		WeaponUUID: &weaponUUID,
	}

	createResult, createResp := createOrder(t, req)
	_ = createResp.Body.Close()
	require.NotNil(t, createResult)

	// Fetch the order and check the optional parts were persisted
	order, resp := getOrder(t, createResult.OrderUUID)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NotNil(t, order.ShieldUUID)
	require.NotNil(t, order.WeaponUUID)
	assert.Equal(t, shieldUUID, *order.ShieldUUID)
	assert.Equal(t, weaponUUID, *order.WeaponUUID)
}

func TestPayment_PayOrder_InvalidUUIDFormat(t *testing.T) {
	_, err := paymentClient.PayOrder(context.Background(), &paymentv1.PayOrderRequest{
		OrderUuid:     "invalid-uuid-format",
		PaymentMethod: paymentv1.PaymentMethod_PAYMENT_METHOD_CARD,
	})
	require.Error(t, err)
	testutil.AssertGRPCStatus(t, err, codes.InvalidArgument)
}

// Full lifecycle tests

func TestOrder_FullLifecycle_CreatePayGet(t *testing.T) {
	// 1. Create the order
	createReq := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullTitaniumUUID,
		EngineUUID: EngineIonBUUID,
		ShieldUUID: new(ShieldEnergyUUID),
	}
	createResult, createResp := createOrder(t, createReq)
	_ = createResp.Body.Close()
	require.NotNil(t, createResult)
	assert.NotEmpty(t, createResult.OrderUUID)

	expectedTotal := int64(HullTitaniumPrice + EngineIonBPrice + ShieldEnergyPrice)
	assert.Equal(t, expectedTotal, createResult.TotalPrice)

	// 2. Fetch the order — expect PENDING_PAYMENT
	order1, getResp1 := getOrder(t, createResult.OrderUUID)
	_ = getResp1.Body.Close()
	assert.Equal(t, "PENDING_PAYMENT", order1.Status)
	assert.Nil(t, order1.TransactionUUID)

	// 3. Pay for the order
	payReq := &PayOrderRequest{PaymentMethod: "SBP"}
	payResult, payResp := payOrder(t, createResult.OrderUUID, payReq)
	_ = payResp.Body.Close()
	require.NotNil(t, payResult)
	assert.NotEmpty(t, payResult.TransactionUUID)

	// 4. Fetch the order — expect PAID
	order2, getResp2 := getOrder(t, createResult.OrderUUID)
	defer func() { _ = getResp2.Body.Close() }()

	assert.Equal(t, "PAID", order2.Status)
	require.NotNil(t, order2.TransactionUUID)
	assert.Equal(t, payResult.TransactionUUID, *order2.TransactionUUID)
	require.NotNil(t, order2.PaymentMethod)
	assert.Equal(t, "SBP", *order2.PaymentMethod)
}

func TestOrder_FullLifecycle_CreateCancelGet(t *testing.T) {
	// 1. Create the order
	createReq := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullAluminumUUID,
		EngineUUID: EngineIonCUUID,
	}
	createResult, createResp := createOrder(t, createReq)
	_ = createResp.Body.Close()
	require.NotNil(t, createResult)

	// 2. Fetch the order — expect PENDING_PAYMENT
	order1, getResp1 := getOrder(t, createResult.OrderUUID)
	_ = getResp1.Body.Close()
	assert.Equal(t, "PENDING_PAYMENT", order1.Status)

	// 3. Cancel the order
	_, cancelResp := cancelOrder(t, createResult.OrderUUID)
	_ = cancelResp.Body.Close()

	// 4. Fetch the order — expect CANCELLED
	order2, getResp2 := getOrder(t, createResult.OrderUUID)
	defer func() { _ = getResp2.Body.Close() }()

	assert.Equal(t, "CANCELLED", order2.Status)
	assert.Nil(t, order2.TransactionUUID)
}

func TestOrder_FullLifecycle_AllPartsPayGet(t *testing.T) {
	// Full lifecycle with all four parts: hull + engine + shield + weapon
	shieldUUID := ShieldEnergyUUID
	weaponUUID := WeaponLaserUUID
	createReq := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullTitaniumUUID,
		EngineUUID: EngineIonBUUID,
		ShieldUUID: &shieldUUID,
		WeaponUUID: &weaponUUID,
	}

	// 1. Create the order
	createResult, createResp := createOrder(t, createReq)
	_ = createResp.Body.Close()
	require.NotNil(t, createResult)

	expectedTotal := int64(HullTitaniumPrice + EngineIonBPrice + ShieldEnergyPrice + WeaponLaserPrice)
	assert.Equal(t, expectedTotal, createResult.TotalPrice)

	// 2. Check every part in the GET response
	order1, getResp1 := getOrder(t, createResult.OrderUUID)
	_ = getResp1.Body.Close()
	assert.Equal(t, HullTitaniumUUID, order1.HullUUID)
	assert.Equal(t, EngineIonBUUID, order1.EngineUUID)
	require.NotNil(t, order1.ShieldUUID)
	assert.Equal(t, shieldUUID, *order1.ShieldUUID)
	require.NotNil(t, order1.WeaponUUID)
	assert.Equal(t, weaponUUID, *order1.WeaponUUID)

	// 3. Pay for the order
	payReq := &PayOrderRequest{PaymentMethod: "CREDIT_CARD"}
	payResult, payResp := payOrder(t, createResult.OrderUUID, payReq)
	_ = payResp.Body.Close()
	require.NotNil(t, payResult)

	// 4. Check the final state
	order2, getResp2 := getOrder(t, createResult.OrderUUID)
	defer func() { _ = getResp2.Body.Close() }()

	assert.Equal(t, "PAID", order2.Status)
	require.NotNil(t, order2.PaymentMethod)
	assert.Equal(t, "CREDIT_CARD", *order2.PaymentMethod)
}

// ogen validation tests (400 Bad Request)

func TestOrder_Create_InvalidBody_EmptyJSON(t *testing.T) {
	httpReq, err := http.NewRequest(http.MethodPost, orderBaseURL()+"/api/v1/orders", bytes.NewReader([]byte("{}")))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")
	withDefaultAuth(httpReq)

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestOrder_Create_InvalidBody_NotJSON(t *testing.T) {
	httpReq, err := http.NewRequest(http.MethodPost, orderBaseURL()+"/api/v1/orders", bytes.NewReader([]byte("not json")))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")
	withDefaultAuth(httpReq)

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestOrder_Create_InvalidBody_MissingHullUUID(t *testing.T) {
	body := `{"engine_uuid": "` + EngineIonCUUID + `"}`
	httpReq, err := http.NewRequest(http.MethodPost, orderBaseURL()+"/api/v1/orders", bytes.NewReader([]byte(body)))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")
	withDefaultAuth(httpReq)

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestOrder_Create_InvalidBody_MissingEngineUUID(t *testing.T) {
	body := `{"hull_uuid": "` + HullAluminumUUID + `"}`
	httpReq, err := http.NewRequest(http.MethodPost, orderBaseURL()+"/api/v1/orders", bytes.NewReader([]byte(body)))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")
	withDefaultAuth(httpReq)

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestOrder_Create_InvalidBody_InvalidHullUUID(t *testing.T) {
	body := `{"hull_uuid": "not-a-uuid", "engine_uuid": "` + EngineIonCUUID + `"}`
	httpReq, err := http.NewRequest(http.MethodPost, orderBaseURL()+"/api/v1/orders", bytes.NewReader([]byte(body)))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")
	withDefaultAuth(httpReq)

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestOrder_Get_InvalidUUIDInPath(t *testing.T) {
	httpReq, err := http.NewRequest(http.MethodGet, orderBaseURL()+"/api/v1/orders/not-a-uuid", nil)
	require.NoError(t, err)
	withDefaultAuth(httpReq)

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestOrder_Pay_InvalidUUIDInPath(t *testing.T) {
	body := `{"payment_method": "CARD"}`
	httpReq, err := http.NewRequest(http.MethodPost, orderBaseURL()+"/api/v1/orders/not-a-uuid/pay", bytes.NewReader([]byte(body)))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")
	withDefaultAuth(httpReq)

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestOrder_Pay_InvalidPaymentMethod(t *testing.T) {
	// Create an order
	createReq := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullAluminumUUID,
		EngineUUID: EngineIonCUUID,
	}
	createResult, createResp := createOrder(t, createReq)
	_ = createResp.Body.Close()
	require.NotNil(t, createResult)

	// Try to pay with an invalid method — ogen rejects it
	body := `{"payment_method": "BITCOIN"}`
	httpReq, err := http.NewRequest(http.MethodPost,
		orderBaseURL()+"/api/v1/orders/"+createResult.OrderUUID+"/pay",
		bytes.NewReader([]byte(body)))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")
	withDefaultAuth(httpReq)

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestOrder_Pay_MissingPaymentMethod(t *testing.T) {
	// Create an order
	createReq := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullAluminumUUID,
		EngineUUID: EngineIonCUUID,
	}
	createResult, createResp := createOrder(t, createReq)
	_ = createResp.Body.Close()
	require.NotNil(t, createResult)

	// Try to pay without payment_method
	body := `{}`
	httpReq, err := http.NewRequest(http.MethodPost,
		orderBaseURL()+"/api/v1/orders/"+createResult.OrderUUID+"/pay",
		bytes.NewReader([]byte(body)))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")
	withDefaultAuth(httpReq)

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestOrder_Pay_EmptyBody(t *testing.T) {
	// Create an order
	createReq := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullAluminumUUID,
		EngineUUID: EngineIonCUUID,
	}
	createResult, createResp := createOrder(t, createReq)
	_ = createResp.Body.Close()
	require.NotNil(t, createResult)

	httpReq, err := http.NewRequest(http.MethodPost,
		orderBaseURL()+"/api/v1/orders/"+createResult.OrderUUID+"/pay",
		bytes.NewReader([]byte("")))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")
	withDefaultAuth(httpReq)

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestOrder_Cancel_InvalidUUIDInPath(t *testing.T) {
	httpReq, err := http.NewRequest(http.MethodPost, orderBaseURL()+"/api/v1/orders/not-a-uuid/cancel", nil)
	require.NoError(t, err)
	withDefaultAuth(httpReq)

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// Out of stock tests

func TestOrder_Create_OutOfStock_Hull(t *testing.T) {
	// The plasma hull has stock_quantity=0, so the order must be rejected
	req := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullOutOfStockUUID,
		EngineUUID: EngineIonCUUID,
	}

	_, resp := createOrder(t, req)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestOrder_Create_OutOfStock_WithOptionalParts(t *testing.T) {
	// An out of stock part among the optional ones — the shield
	req := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullAluminumUUID,
		EngineUUID: EngineIonCUUID,
		ShieldUUID: new(HullOutOfStockUUID), // a hull UUID passed as a shield: the type mismatches, but out of stock is checked first
	}

	_, resp := createOrder(t, req)
	defer func() { _ = resp.Body.Close() }()

	// Either Conflict (out of stock) or another error — but not 201.
	assert.NotEqual(t, http.StatusCreated, resp.StatusCode)
}

// Inventory tests: an out of stock part is still listed

func TestInventory_GetPart_OutOfStock(t *testing.T) {
	resp, err := inventoryClient.GetPart(authCtx(context.Background()), &inventoryv1.GetPartRequest{
		Uuid: HullOutOfStockUUID,
	})
	require.NoError(t, err)

	part := resp.GetPart()
	assert.Equal(t, HullOutOfStockUUID, part.GetUuid())
	assert.Equal(t, int64(HullOutOfStockPrice), part.GetPrice())
	assert.Equal(t, inventoryv1.PartType_PART_TYPE_HULL, part.GetPartType())
	assert.Equal(t, int64(0), part.GetStockQuantity())
	assert.Equal(t, "Experimental hull (out of stock)", part.GetDescription())
}

func TestInventory_ListParts_ByUuids_IncludesOutOfStock(t *testing.T) {
	uuids := []string{HullAluminumUUID, HullOutOfStockUUID}

	resp, err := inventoryClient.ListParts(authCtx(context.Background()), &inventoryv1.ListPartsRequest{
		Uuids: uuids,
	})
	require.NoError(t, err)
	assert.Len(t, resp.GetParts(), 2)

	// The out of stock part is returned: there is no availability filter
	assert.Equal(t, HullOutOfStockUUID, resp.GetParts()[1].GetUuid())
	assert.Equal(t, int64(0), resp.GetParts()[1].GetStockQuantity())
}

// Order tests: created_at

func TestOrder_Get_VerifyCreatedAt(t *testing.T) {
	createReq := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullAluminumUUID,
		EngineUUID: EngineIonCUUID,
	}
	createResult, createResp := createOrder(t, createReq)
	_ = createResp.Body.Close()
	require.NotNil(t, createResult)

	order, resp := getOrder(t, createResult.OrderUUID)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NotEmpty(t, order.CreatedAt, "created_at must not be empty")

	// Parse the timestamp: the string must be valid and the time non-zero
	createdAt, err := time.Parse(time.RFC3339Nano, order.CreatedAt)
	if err != nil {
		createdAt, err = time.Parse(time.RFC3339, order.CreatedAt)
	}
	if err != nil {
		createdAt, err = time.Parse("2006-01-02T15:04:05Z", order.CreatedAt)
	}
	require.NoError(t, err, "failed to parse created_at: %s", order.CreatedAt)
	assert.False(t, createdAt.IsZero(), "created_at must not be zero")
}

// Tests with a shield only (no weapon)

func TestOrder_Create_WithShieldOnly(t *testing.T) {
	req := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullAluminumUUID,
		EngineUUID: EngineIonCUUID,
		ShieldUUID: new(ShieldEnergyUUID),
	}

	result, resp := createOrder(t, req)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.NotNil(t, result)
	expectedTotal := int64(HullAluminumPrice + EngineIonCPrice + ShieldEnergyPrice)
	assert.Equal(t, expectedTotal, result.TotalPrice)
}

// Tests matching part types to ship slots

func TestOrder_Create_WrongPartType_WeaponAsHull(t *testing.T) {
	// A weapon UUID passed into the hull slot: InventoryService returns InvalidArgument
	// and the order service maps it to 400 Bad Request
	req := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   WeaponLaserUUID,
		EngineUUID: EngineIonCUUID,
	}

	_, resp := createOrder(t, req)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestOrder_Create_WrongPartType_HullAsEngine(t *testing.T) {
	// A hull UUID passed into the engine slot (effectively a second hull) — 400
	req := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullAluminumUUID,
		EngineUUID: HullTitaniumUUID,
	}

	_, resp := createOrder(t, req)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestOrder_Create_WrongPartType_ShieldAsWeapon(t *testing.T) {
	// A shield UUID passed into the weapon slot — 400
	req := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullAluminumUUID,
		EngineUUID: EngineIonCUUID,
		WeaponUUID: new(ShieldEnergyUUID),
	}

	_, resp := createOrder(t, req)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestOrder_Create_DuplicateUUID_HullAndEngine(t *testing.T) {
	// The same UUID in hull and engine automatically means a type mismatch in one of the
	// slots (a part cannot be both HULL and ENGINE) → 400
	req := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullAluminumUUID,
		EngineUUID: HullAluminumUUID,
	}

	_, resp := createOrder(t, req)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ValidateCompatibility tests (gRPC)

func TestInventory_ValidateCompatibility_Success_Compatible(t *testing.T) {
	// Aluminium hull (strength=50) + ion engine C (required_strength=30) — compatible
	_, err := inventoryClient.ValidateCompatibility(authCtx(context.Background()), &inventoryv1.ValidateCompatibilityRequest{
		HullUuid:   HullAluminumUUID,
		EngineUuid: EngineIonCUUID,
	})
	require.NoError(t, err)
}

func TestInventory_ValidateCompatibility_Success_StrongHull(t *testing.T) {
	// Titanium hull (strength=150) + ion engine B (required_strength=70) — compatible
	_, err := inventoryClient.ValidateCompatibility(authCtx(context.Background()), &inventoryv1.ValidateCompatibilityRequest{
		HullUuid:   HullTitaniumUUID,
		EngineUuid: EngineIonBUUID,
	})
	require.NoError(t, err)
}

func TestInventory_ValidateCompatibility_Success_AllParts(t *testing.T) {
	// Titanium hull + Ion B + energy shield + laser — all compatible
	_, err := inventoryClient.ValidateCompatibility(authCtx(context.Background()), &inventoryv1.ValidateCompatibilityRequest{
		HullUuid:   HullTitaniumUUID,
		EngineUuid: EngineIonBUUID,
		ShieldUuid: ShieldEnergyUUID,
		WeaponUuid: WeaponLaserUUID,
	})
	require.NoError(t, err)
}

func TestInventory_ValidateCompatibility_Fail_WeakHull(t *testing.T) {
	// Aluminium hull (strength=50) + ion engine B (required_strength=70) — incompatible
	// the hull is too weak for a class B engine
	_, err := inventoryClient.ValidateCompatibility(authCtx(context.Background()), &inventoryv1.ValidateCompatibilityRequest{
		HullUuid:   HullAluminumUUID,
		EngineUuid: EngineIonBUUID,
	})
	require.Error(t, err)
	testutil.AssertGRPCStatus(t, err, codes.FailedPrecondition)
}

func TestInventory_ValidateCompatibility_MissingHull(t *testing.T) {
	// Without hull_uuid the contract is violated (the slot is required)
	_, err := inventoryClient.ValidateCompatibility(authCtx(context.Background()), &inventoryv1.ValidateCompatibilityRequest{
		EngineUuid: EngineIonBUUID,
	})
	require.Error(t, err)
	testutil.AssertGRPCStatus(t, err, codes.InvalidArgument)
}

func TestInventory_ValidateCompatibility_MissingEngine(t *testing.T) {
	_, err := inventoryClient.ValidateCompatibility(authCtx(context.Background()), &inventoryv1.ValidateCompatibilityRequest{
		HullUuid: HullAluminumUUID,
	})
	require.Error(t, err)
	testutil.AssertGRPCStatus(t, err, codes.InvalidArgument)
}

func TestInventory_ValidateCompatibility_TypeMismatch_WeaponInHullSlot(t *testing.T) {
	// A weapon UUID passed into the hull slot — InvalidArgument
	_, err := inventoryClient.ValidateCompatibility(authCtx(context.Background()), &inventoryv1.ValidateCompatibilityRequest{
		HullUuid:   WeaponLaserUUID,
		EngineUuid: EngineIonCUUID,
	})
	require.Error(t, err)
	testutil.AssertGRPCStatus(t, err, codes.InvalidArgument)
}

func TestInventory_ValidateCompatibility_TypeMismatch_HullInEngineSlot(t *testing.T) {
	// A hull UUID passed into the engine slot (effectively a second hull) — InvalidArgument
	_, err := inventoryClient.ValidateCompatibility(authCtx(context.Background()), &inventoryv1.ValidateCompatibilityRequest{
		HullUuid:   HullAluminumUUID,
		EngineUuid: HullTitaniumUUID,
	})
	require.Error(t, err)
	testutil.AssertGRPCStatus(t, err, codes.InvalidArgument)
}

func TestInventory_ValidateCompatibility_DuplicateUUID_HullAndEngine(t *testing.T) {
	// The same UUID in two slots — InvalidArgument
	_, err := inventoryClient.ValidateCompatibility(authCtx(context.Background()), &inventoryv1.ValidateCompatibilityRequest{
		HullUuid:   HullAluminumUUID,
		EngineUuid: HullAluminumUUID,
	})
	require.Error(t, err)
	testutil.AssertGRPCStatus(t, err, codes.InvalidArgument)
}

func TestInventory_ValidateCompatibility_NotFound(t *testing.T) {
	// A non-existent UUID — NotFound
	_, err := inventoryClient.ValidateCompatibility(authCtx(context.Background()), &inventoryv1.ValidateCompatibilityRequest{
		HullUuid:   HullAluminumUUID,
		EngineUuid: uuid.New().String(),
	})
	require.Error(t, err)
	testutil.AssertGRPCStatus(t, err, codes.NotFound)
}

// ReserveParts tests (gRPC)

func TestInventory_ReserveParts_Success(t *testing.T) {
	// Reserve available parts — success expected
	_, err := inventoryClient.ReserveParts(authCtx(context.Background()), &inventoryv1.ReservePartsRequest{
		Uuids: []string{HullAluminumUUID, EngineIonCUUID},
	})
	require.NoError(t, err)

	// Release them again so the other tests are unaffected
	_, err = inventoryClient.ReleaseParts(authCtx(context.Background()), &inventoryv1.ReleasePartsRequest{
		Uuids: []string{HullAluminumUUID, EngineIonCUUID},
	})
	require.NoError(t, err)
}

func TestInventory_ReserveParts_OutOfStock(t *testing.T) {
	// The plasma hull (stock=0) cannot be reserved
	_, err := inventoryClient.ReserveParts(authCtx(context.Background()), &inventoryv1.ReservePartsRequest{
		Uuids: []string{HullOutOfStockUUID},
	})
	require.Error(t, err)
	testutil.AssertGRPCStatus(t, err, codes.ResourceExhausted)
}

func TestInventory_ReserveParts_NotFound(t *testing.T) {
	_, err := inventoryClient.ReserveParts(authCtx(context.Background()), &inventoryv1.ReservePartsRequest{
		Uuids: []string{uuid.New().String()},
	})
	require.Error(t, err)
	testutil.AssertGRPCStatus(t, err, codes.NotFound)
}

func TestInventory_ReserveParts_SinglePart(t *testing.T) {
	// Reserve a single part
	_, err := inventoryClient.ReserveParts(authCtx(context.Background()), &inventoryv1.ReservePartsRequest{
		Uuids: []string{ShieldEnergyUUID},
	})
	require.NoError(t, err)

	// Release it again
	_, err = inventoryClient.ReleaseParts(authCtx(context.Background()), &inventoryv1.ReleasePartsRequest{
		Uuids: []string{ShieldEnergyUUID},
	})
	require.NoError(t, err)
}

// ReleaseParts tests (gRPC)

func TestInventory_ReleaseParts_Success(t *testing.T) {
	// Reserve first, then release — the full cycle
	uuids := []string{HullTitaniumUUID, EngineIonBUUID}

	_, err := inventoryClient.ReserveParts(authCtx(context.Background()), &inventoryv1.ReservePartsRequest{
		Uuids: uuids,
	})
	require.NoError(t, err)

	_, err = inventoryClient.ReleaseParts(authCtx(context.Background()), &inventoryv1.ReleasePartsRequest{
		Uuids: uuids,
	})
	require.NoError(t, err)
}

func TestInventory_ReleaseParts_NothingToRelease(t *testing.T) {
	// The plasma hull (stock=0, reserved=0) — nothing to release
	_, err := inventoryClient.ReleaseParts(authCtx(context.Background()), &inventoryv1.ReleasePartsRequest{
		Uuids: []string{HullOutOfStockUUID},
	})
	require.Error(t, err)
	testutil.AssertGRPCStatus(t, err, codes.FailedPrecondition)
}

func TestInventory_ReleaseParts_NotFound(t *testing.T) {
	_, err := inventoryClient.ReleaseParts(authCtx(context.Background()), &inventoryv1.ReleasePartsRequest{
		Uuids: []string{uuid.New().String()},
	})
	require.Error(t, err)
	testutil.AssertGRPCStatus(t, err, codes.NotFound)
}

// Order Create tests with incompatible parts (HTTP)

func TestOrder_Create_IncompatibleParts_WeakHullStrongEngine(t *testing.T) {
	// Aluminium hull (strength=50) + ion engine B (required_strength=70).
	// The hull cannot support the engine, so ValidateCompatibility returns
	// FailedPrecondition and the order service turns it into 409 Conflict
	req := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullAluminumUUID,
		EngineUUID: EngineIonBUUID,
	}

	_, resp := createOrder(t, req)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestOrder_Create_IncompatibleParts_WithOptionalParts(t *testing.T) {
	// Aluminium hull (strength=50) + ion engine B (required_strength=70). + Shield + Weapon
	// Even with the optional parts, the hull/engine mismatch blocks creation
	req := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullAluminumUUID,
		EngineUUID: EngineIonBUUID,
		ShieldUUID: new(ShieldEnergyUUID),
		WeaponUUID: new(WeaponLaserUUID),
	}

	_, resp := createOrder(t, req)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestOrder_Create_CompatibleParts_StrongHullStrongEngine(t *testing.T) {
	// Titanium hull (strength=150) + ion engine B (required_strength=70) — compatible
	// Control test: with compatible parts the order is created
	req := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullTitaniumUUID,
		EngineUUID: EngineIonBUUID,
	}

	result, resp := createOrder(t, req)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.NotNil(t, result)
	assert.Equal(t, int64(HullTitaniumPrice+EngineIonBPrice), result.TotalPrice)
}

// Reserve/Release tests across the full order lifecycle

func TestInventory_ReserveRelease_FullCycle(t *testing.T) {
	// Reserve → release → reserve again, checking the counters stay consistent
	uuids := []string{WeaponLaserUUID}

	// First reservation
	_, err := inventoryClient.ReserveParts(authCtx(context.Background()), &inventoryv1.ReservePartsRequest{
		Uuids: uuids,
	})
	require.NoError(t, err)

	// Release
	_, err = inventoryClient.ReleaseParts(authCtx(context.Background()), &inventoryv1.ReleasePartsRequest{
		Uuids: uuids,
	})
	require.NoError(t, err)

	// The second reservation must succeed (the part is available again)
	_, err = inventoryClient.ReserveParts(authCtx(context.Background()), &inventoryv1.ReservePartsRequest{
		Uuids: uuids,
	})
	require.NoError(t, err)

	// Final release
	_, err = inventoryClient.ReleaseParts(authCtx(context.Background()), &inventoryv1.ReleasePartsRequest{
		Uuids: uuids,
	})
	require.NoError(t, err)
}

// user_uuid tests (propagated through the whole chain from the authenticated session)

func TestOrder_Get_ReturnsUserUUID(t *testing.T) {
	// Since week 6 user_uuid comes from the authenticated session, not the request body.
	// Register a new user and log in to obtain a dedicated session, create an order under
	// it and check that order.UserUUID matches this user's UUID.
	//
	sessionUUID, userUUID := registerAndLogin(t, "owner-"+uuid.New().String()[:8], "password123")

	body := `{"hull_uuid": "` + HullAluminumUUID + `", "engine_uuid": "` + EngineIonCUUID + `"}`
	createReq, err := http.NewRequest(http.MethodPost, orderBaseURL()+"/api/v1/orders", bytes.NewReader([]byte(body)))
	require.NoError(t, err)
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+sessionUUID)

	createResp, err := httpClient.Do(createReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)

	var created CreateOrderResponse
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))
	_ = createResp.Body.Close()

	getReq, err := http.NewRequest(http.MethodGet, orderBaseURL()+"/api/v1/orders/"+created.OrderUUID, nil)
	require.NoError(t, err)
	getReq.Header.Set("Authorization", "Bearer "+sessionUUID)

	getResp, err := httpClient.Do(getReq)
	require.NoError(t, err)
	defer func() { _ = getResp.Body.Close() }()

	require.Equal(t, http.StatusOK, getResp.StatusCode)
	var order OrderDTO
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&order))
	assert.Equal(t, userUUID, order.UserUUID, "user_uuid must come from the authenticated session")
}

// Cancel tests for the ASSEMBLED status
//
// API tests have no Kafka (noopProducer), so ASSEMBLED cannot be reached through the
// usual Pay → OrderPaid → AssemblyService → ShipAssembled chain. The status is set
// directly in the database, which still tests the Cancel logic honestly

func TestOrder_Cancel_AlreadyAssembled(t *testing.T) {
	// Create and pay for an order so that the parts are reserved
	createReq := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullAluminumUUID,
		EngineUUID: EngineIonCUUID,
	}
	createResult, createResp := createOrder(t, createReq)
	_ = createResp.Body.Close()
	require.NotNil(t, createResult)

	payReq := &PayOrderRequest{PaymentMethod: "CARD"}
	_, payResp := payOrder(t, createResult.OrderUUID, payReq)
	_ = payResp.Body.Close()

	// Simulate the finished assembly: move the order to ASSEMBLED bypassing the Kafka chain
	_, err := orderDBPool.Exec(
		context.Background(),
		`UPDATE orders SET status = 'ASSEMBLED' WHERE uuid = $1`,
		createResult.OrderUUID,
	)
	require.NoError(t, err)

	// Cancelling an assembled order must return 409 Conflict
	_, cancelResp := cancelOrder(t, createResult.OrderUUID)
	defer func() { _ = cancelResp.Body.Close() }()

	require.Equal(t, http.StatusConflict, cancelResp.StatusCode)

	// The order status must not change: it stays ASSEMBLED
	order, getResp := getOrder(t, createResult.OrderUUID)
	_ = getResp.Body.Close()
	assert.Equal(t, "ASSEMBLED", order.Status)
}

// CommitParts tests (gRPC)

func TestInventory_CommitParts_Success(t *testing.T) {
	// Full cycle: reserve → commit. After the commit stock must drop by 1.
	uuids := []string{ShieldEnergyUUID}

	partBefore, err := inventoryClient.GetPart(authCtx(context.Background()), &inventoryv1.GetPartRequest{
		Uuid: ShieldEnergyUUID,
	})
	require.NoError(t, err)
	stockBefore := partBefore.GetPart().GetStockQuantity()

	_, err = inventoryClient.ReserveParts(authCtx(context.Background()), &inventoryv1.ReservePartsRequest{
		Uuids: uuids,
	})
	require.NoError(t, err)

	_, err = inventoryClient.CommitParts(authCtx(context.Background()), &inventoryv1.CommitPartsRequest{
		Uuids: uuids,
	})
	require.NoError(t, err)

	partAfter, err := inventoryClient.GetPart(authCtx(context.Background()), &inventoryv1.GetPartRequest{
		Uuid: ShieldEnergyUUID,
	})
	require.NoError(t, err)
	assert.Equal(t, stockBefore-1, partAfter.GetPart().GetStockQuantity(),
		"Commit must decrease stock_quantity by 1")
}

func TestInventory_CommitParts_NothingToCommit(t *testing.T) {
	// The plasma hull exists but has stock=0 and reserved=0, so there is nothing to commit.
	// ListForUpdate finds the part, but the SQL condition stock>0 AND reserved>0 fails
	// RowsAffected=0 → ErrNothingToCommit → FailedPrecondition
	_, err := inventoryClient.CommitParts(authCtx(context.Background()), &inventoryv1.CommitPartsRequest{
		Uuids: []string{HullOutOfStockUUID},
	})
	require.Error(t, err)
	testutil.AssertGRPCStatus(t, err, codes.FailedPrecondition)
}

func TestInventory_CommitParts_NotFound(t *testing.T) {
	// A non-existent UUID → ListForUpdate returns ErrPartNotFound → NotFound.
	// This guards Commit itself: "no such part" and "nothing to commit" stay distinct
	_, err := inventoryClient.CommitParts(authCtx(context.Background()), &inventoryv1.CommitPartsRequest{
		Uuids: []string{uuid.New().String()},
	})
	require.Error(t, err)
	testutil.AssertGRPCStatus(t, err, codes.NotFound)
}

func TestInventory_CommitParts_PartialCommit_RollbackOnMissing(t *testing.T) {
	// If one part in the batch is valid (reserved) and another is not, the whole Commit
	// must roll back: the stock of the first part must stay unchanged
	validUUID := HullTitaniumUUID

	partBefore, err := inventoryClient.GetPart(authCtx(context.Background()), &inventoryv1.GetPartRequest{
		Uuid: validUUID,
	})
	require.NoError(t, err)
	stockBefore := partBefore.GetPart().GetStockQuantity()

	_, err = inventoryClient.ReserveParts(authCtx(context.Background()), &inventoryv1.ReservePartsRequest{
		Uuids: []string{validUUID},
	})
	require.NoError(t, err)

	// A batch with a valid and a non-existent part → FailedPrecondition, the transaction rolls back
	_, err = inventoryClient.CommitParts(authCtx(context.Background()), &inventoryv1.CommitPartsRequest{
		Uuids: []string{validUUID, uuid.New().String()},
	})
	require.Error(t, err)

	partAfter, err := inventoryClient.GetPart(authCtx(context.Background()), &inventoryv1.GetPartRequest{
		Uuid: validUUID,
	})
	require.NoError(t, err)
	assert.Equal(t, stockBefore, partAfter.GetPart().GetStockQuantity(),
		"on a partial failure the stock of the valid part must stay unchanged")

	// Clean the reservation up so the neighbouring tests are unaffected
	_, err = inventoryClient.ReleaseParts(authCtx(context.Background()), &inventoryv1.ReleasePartsRequest{
		Uuids: []string{validUUID},
	})
	require.NoError(t, err)
}

// Cancel tests: returning the reservation

func TestOrder_Cancel_ReleasesReservedParts(t *testing.T) {
	// After Cancel the reserved parts must be released.
	// Before Cancel reserved was +1; afterwards it must return to the original value
	partBefore, err := inventoryClient.GetPart(authCtx(context.Background()), &inventoryv1.GetPartRequest{
		Uuid: HullTitaniumUUID,
	})
	require.NoError(t, err)
	stockBefore := partBefore.GetPart().GetStockQuantity()

	createReq := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullTitaniumUUID,
		EngineUUID: EngineIonBUUID,
	}
	createResult, createResp := createOrder(t, createReq)
	_ = createResp.Body.Close()
	require.NotNil(t, createResult)

	_, cancelResp := cancelOrder(t, createResult.OrderUUID)
	defer func() { _ = cancelResp.Body.Close() }()
	require.Equal(t, http.StatusOK, cancelResp.StatusCode)

	// Cancel must not change stock (Reserve does not touch stock, and Release only
	// returns reserved to its original value)
	partAfter, err := inventoryClient.GetPart(authCtx(context.Background()), &inventoryv1.GetPartRequest{
		Uuid: HullTitaniumUUID,
	})
	require.NoError(t, err)
	assert.Equal(t, stockBefore, partAfter.GetPart().GetStockQuantity(),
		"Cancel must not consume stock, only release the reservation")
}

// Concurrency tests (SELECT FOR UPDATE)
//
// These tests prove the pessimistic locks really work: without FOR UPDATE two parallel
// requests would see the same state and both succeed — a race condition. With FOR
// UPDATE the second request waits for the first, sees the updated state and correctly
// rejects.

func TestOrder_Pay_Concurrent_SameOrder(t *testing.T) {
	// Two parallel Pay calls on the same order.
	// FOR UPDATE in OrderRepo.GetForUpdate guarantees exactly one Pay returns 200 and the
	// other sees status PAID and returns 409 (already paid).
	// Without FOR UPDATE both would succeed → a double payment.
	createReq := &CreateOrderRequest{
		UserUUID:   uuid.New().String(),
		HullUUID:   HullAluminumUUID,
		EngineUUID: EngineIonCUUID,
	}
	createResult, createResp := createOrder(t, createReq)
	_ = createResp.Body.Close()
	require.NotNil(t, createResult)

	var wg sync.WaitGroup
	statusCodes := make([]int, 2)
	for i := range 2 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			payReq := &PayOrderRequest{PaymentMethod: "CARD"}
			_, resp := payOrder(t, createResult.OrderUUID, payReq)
			defer func() { _ = resp.Body.Close() }()
			statusCodes[idx] = resp.StatusCode
		}(i)
	}
	wg.Wait()

	successCount := 0
	conflictCount := 0
	for _, code := range statusCodes {
		switch code {
		case http.StatusOK:
			successCount++
		case http.StatusConflict:
			conflictCount++
		}
	}

	assert.Equal(t, 1, successCount, "exactly one Pay must return 200")
	assert.Equal(t, 1, conflictCount, "exactly one Pay must return 409 (already paid)")
}

func TestInventory_ReserveParts_Concurrent_LastPart(t *testing.T) {
	// Prepare a part with stock_quantity=1 — the "last one in stock".
	// Two parallel ReserveParts calls on the same part: FOR UPDATE in ListForUpdate
	// guarantees one reservation succeeds and the other fails with FailedPrecondition
	// (OutOfStock).
	// Without FOR UPDATE both would reserve → the same part reserved twice.
	testPartUUID := uuid.New().String()
	_, err := inventoryDBPool.Exec(
		context.Background(),
		`INSERT INTO parts (uuid, name, description, part_type, price, stock_quantity, properties)
         VALUES ($1, 'Test hull', 'Concurrency test', 'HULL', 1000, 1, '{"hull": {"strength": 100}}')`,
		testPartUUID,
	)
	require.NoError(t, err)
	defer func() {
		_, _ = inventoryDBPool.Exec(
			context.Background(),
			`DELETE FROM parts WHERE uuid = $1`, testPartUUID,
		)
	}()

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := range 2 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(authCtx(context.Background()), 10*time.Second)
			defer cancel()
			_, errs[idx] = inventoryClient.ReserveParts(ctx, &inventoryv1.ReservePartsRequest{
				Uuids: []string{testPartUUID},
			})
		}(i)
	}
	wg.Wait()

	successCount := 0
	failedCount := 0
	for _, e := range errs {
		if e == nil {
			successCount++
		} else {
			failedCount++
		}
	}

	assert.Equal(t, 1, successCount, "exactly one Reserve must succeed")
	assert.Equal(t, 1, failedCount, "exactly one Reserve must fail (out of stock)")
}

// OrderService HTTP middleware tests (auth)
//
// Every Order endpoint requires a valid Bearer session: the middleware calls
// AuthService.Whoami and answers 401 with a plain text body on any error

func TestAuthMiddleware_NoAuthorizationHeader(t *testing.T) {
	body := `{"hull_uuid": "` + HullAluminumUUID + `", "engine_uuid": "` + EngineIonCUUID + `"}`
	httpReq, err := http.NewRequest(http.MethodPost, orderBaseURL()+"/api/v1/orders", bytes.NewReader([]byte(body)))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")
	// No Authorization header

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestAuthMiddleware_BadAuthorizationFormat(t *testing.T) {
	body := `{"hull_uuid": "` + HullAluminumUUID + `", "engine_uuid": "` + EngineIonCUUID + `"}`
	httpReq, err := http.NewRequest(http.MethodPost, orderBaseURL()+"/api/v1/orders", bytes.NewReader([]byte(body)))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Token abc") // wrong prefix

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestAuthMiddleware_InvalidSession(t *testing.T) {
	body := `{"hull_uuid": "` + HullAluminumUUID + `", "engine_uuid": "` + EngineIonCUUID + `"}`
	httpReq, err := http.NewRequest(http.MethodPost, orderBaseURL()+"/api/v1/orders", bytes.NewReader([]byte(body)))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+uuid.New().String())

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestAuthMiddleware_ExpiredSession(t *testing.T) {
	// Register, log in, then log out — simulating an expired session
	sessionUUID, _ := registerAndLogin(t, "expired-"+uuid.New().String()[:8], "password123")
	_, err := authSvcClient.Logout(context.Background(), &authv1.LogoutRequest{SessionUuid: sessionUUID})
	require.NoError(t, err)

	body := `{"hull_uuid": "` + HullAluminumUUID + `", "engine_uuid": "` + EngineIonCUUID + `"}`
	httpReq, err := http.NewRequest(http.MethodPost, orderBaseURL()+"/api/v1/orders", bytes.NewReader([]byte(body)))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+sessionUUID)

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// InventoryService gRPC interceptor tests (auth)
//
// Every Inventory method requires session-uuid in the incoming metadata;
// a missing or invalid session yields codes.Unauthenticated

func TestInterceptor_NoMetadata(t *testing.T) {
	// A direct call without metadata — Unauthenticated
	_, err := inventoryClient.GetPart(context.Background(), &inventoryv1.GetPartRequest{
		Uuid: HullAluminumUUID,
	})
	require.Error(t, err)
	testutil.AssertGRPCStatus(t, err, codes.Unauthenticated)
}

func TestInterceptor_EmptySession(t *testing.T) {
	ctx := authCtxWith(context.Background(), "")
	_, err := inventoryClient.GetPart(ctx, &inventoryv1.GetPartRequest{
		Uuid: HullAluminumUUID,
	})
	require.Error(t, err)
	testutil.AssertGRPCStatus(t, err, codes.Unauthenticated)
}

func TestInterceptor_InvalidSession(t *testing.T) {
	ctx := authCtxWith(context.Background(), uuid.New().String())
	_, err := inventoryClient.GetPart(ctx, &inventoryv1.GetPartRequest{
		Uuid: HullAluminumUUID,
	})
	require.Error(t, err)
	testutil.AssertGRPCStatus(t, err, codes.Unauthenticated)
}
