//go:build e2e

// Package e2e holds the end-to-end tests of OrderService against a real Kafka (Redpanda).
//
// The suite exists to exercise the asynchronous chain:
//
//	HTTP Pay → orderProducer (Kafka topic order.paid)
//	         → the real AssemblyService
//	         → Kafka topic assembly.ship-assembled
//	         → order/internal/consumer/assembly_consumer
//	         → CommitParts + UPDATE orders SET status=ASSEMBLED
//
// Since week 6 the chain is session-authenticated: HTTP is guarded by the Bearer
// middleware, Inventory gRPC by an auth interceptor, and session_uuid travels through
// Kafka headers (see platform/pkg/middleware/kafka). That is why the setup also starts
// IAM (Postgres + Redis + a bufconn server) and gives the Inventory bufconn client a
//
// SessionForwarder. The suite only builds under the e2e build tag, so a plain
// go test ./... does not pay the Redpanda startup cost.
package e2e

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/IBM/sarama"
	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	tcredpanda "github.com/testcontainers/testcontainers-go/modules/redpanda"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	assemblyApp "github.com/vixart/rocket-factory/assembly/pkg/app"
	iamApp "github.com/vixart/rocket-factory/iam/pkg/app"
	invApp "github.com/vixart/rocket-factory/inventory/pkg/app"
	inventoryClientPkg "github.com/vixart/rocket-factory/order/internal/client/grpc/inventory/v1"
	assemblyconsumer "github.com/vixart/rocket-factory/order/internal/consumer/assembly_consumer"
	"github.com/vixart/rocket-factory/order/internal/interceptor"
	orderProducer "github.com/vixart/rocket-factory/order/internal/producer/order_producer"
	orderRepoPkg "github.com/vixart/rocket-factory/order/internal/repository/order"
	"github.com/vixart/rocket-factory/order/pkg/app"
	payApp "github.com/vixart/rocket-factory/payment/pkg/app"
	wrappedKafkaConsumer "github.com/vixart/rocket-factory/platform/pkg/kafka/consumer"
	wrappedKafkaProducer "github.com/vixart/rocket-factory/platform/pkg/kafka/producer"
	kafkaMiddleware "github.com/vixart/rocket-factory/platform/pkg/middleware/kafka"
	authv1 "github.com/vixart/rocket-factory/shared/pkg/proto/auth/v1"
	inventoryv1 "github.com/vixart/rocket-factory/shared/pkg/proto/inventory/v1"
	paymentv1 "github.com/vixart/rocket-factory/shared/pkg/proto/payment/v1"
	userv1 "github.com/vixart/rocket-factory/shared/pkg/proto/user/v1"
)

// Seeded part UUIDs and prices (from migrations/inventory/00002_seed_parts.sql).
// They are duplicated from tests/api_test.go on purpose: e2e is a standalone suite
// that must not depend on a neighbouring package.
const (
	HullAluminumUUID = "550e8400-e29b-41d4-a716-446655440001" // 500000 kopecks
	EngineIonCUUID   = "550e8400-e29b-41d4-a716-446655440003" // 300000 kopecks

	HullAluminumPrice = 500000
	EngineIonCPrice   = 300000
)

const (
	bufSize = 1024 * 1024

	// Redpanda takes ~5-10 seconds to start on macOS, leave some margin
	redpandaImage = "docker.redpanda.com/redpandadata/redpanda:v25.1.7"

	// Redis for IAM sessions — the version matches solutions/week_6/iam.env
	redisImage = "redis:8.6.3-alpine3.23"

	// numPartitions=1 is enough: the test cares about delivery, not scaling
	topicPartitions        = 1
	topicReplicationFactor = 1

	// sessionTTL: an hour is plenty for e2e — the test is short, the session cannot
	// expire, and no re-login happens between steps
	sessionTTL = time.Hour
)

var (
	httpClient = &http.Client{Timeout: 10 * time.Second}
	ts         *httptest.Server

	// Unique topics and group ids per run isolate parallel CI builds that may share
	// the same Redpanda cluster
	orderPaidTopic     string
	shipAssembledTopic string
	assemblyGroupID    string
	orderGroupID       string

	// A direct pool to the order database, needed in the rare cases where state is not
	// exposed through the API (auditing specific fields, for example)
	orderDBPool     *pgxpool.Pool
	inventoryDBPool *pgxpool.Pool

	inventoryClient inventoryv1.InventoryServiceClient

	// The IAM clients are exported for lifecycle_test, which performs Register/Login to
	// obtain the sessionUUID for the Bearer header
	authSvcClient authv1.AuthServiceClient
	userSvcClient userv1.UserServiceClient
)

// runMain wraps m.Run so that the deferred cleanup runs even if the setup panics.
// os.Exit skips defers, so Exit is called separately from TestMain.
func TestMain(m *testing.M) {
	os.Exit(runMain(m))
}

func runMain(m *testing.M) int {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cleanups := newCleanupStack()
	defer cleanups.run(context.Background())

	// 1-3. PostgreSQL × 3 (order + inventory + iam) — three separate containers
	orderPool := startPostgresAndMigrate(ctx, cleanups, "order-service", "../../../migrations/order")
	inventoryPool := startPostgresAndMigrate(ctx, cleanups, "inventory-service", "../../../migrations/inventory")
	iamPool := startPostgresAndMigrate(ctx, cleanups, "iam-service", "../../../migrations/iam")
	orderDBPool = orderPool
	inventoryDBPool = inventoryPool

	txManager := mustNew(manager.New(trmpgx.NewDefaultFactory(orderPool)))

	// 4. Redis — the IAM session storage
	redisClient := startRedis(ctx, cleanups)

	// 5. IAM gRPC over bufconn — needed by the Inventory server (auth interceptor), by
	// the Order HTTP handler (middleware) and by the test itself (Register/Login).
	// bcrypt.MinCost is critical, otherwise the test becomes several times slower
	iamConn := startBufconnGRPCIAM(ctx, cleanups, iamPool, redisClient)
	authSvcClient = authv1.NewAuthServiceClient(iamConn)
	userSvcClient = userv1.NewUserServiceClient(iamConn)

	// 6. Inventory + Payment gRPC over bufconn — Kafka does not touch them, so they stay
	// in memory (faster than starting more containers). Inventory carries an auth
	// interceptor on the server and a SessionForwarder on the client, otherwise
	// CommitParts from assembly_consumer and GetPart from the test would fail
	// authentication
	invConn := startBufconnGRPCInventory(ctx, cleanups, inventoryPool, authSvcClient)
	payConn := startBufconnGRPCPayment(ctx, cleanups)

	inventoryClient = inventoryv1.NewInventoryServiceClient(invConn)
	paymentClient := paymentv1.NewPaymentServiceClient(payConn)

	// 7. Redpanda — real Kafka-compatible infrastructure
	broker := startRedpanda(ctx, cleanups)

	// 8. Unique topics per run, created explicitly through the AdminClient. A sarama
	// consumer fails when the topic does not exist yet, so auto-create cannot be relied
	// upon — the topics must be ready first
	suffix := time.Now().UnixNano()
	orderPaidTopic = fmt.Sprintf("e2e-%d-order.paid", suffix)
	shipAssembledTopic = fmt.Sprintf("e2e-%d-assembly.ship-assembled", suffix)
	assemblyGroupID = fmt.Sprintf("e2e-%d-assembly-service", suffix)
	orderGroupID = fmt.Sprintf("e2e-%d-order-service", suffix)
	createTopics(broker, orderPaidTopic, shipAssembledTopic)

	// 9. A real sarama producer for order — publishes OrderPaid to Kafka
	syncProducer := mustNew(sarama.NewSyncProducer([]string{broker}, producerConfig()))
	cleanups.add("order sarama producer", func(_ context.Context) error { return syncProducer.Close() })

	orderPaidKafkaProducer := wrappedKafkaProducer.NewProducer(syncProducer, orderPaidTopic)
	realOrderProducer := orderProducer.New(orderPaidKafkaProducer)

	// 10. The Order HTTP server with the real producer (NOT the noopProducer of api_test).
	// authSvcClient is wired in as well: since week 6 the HTTP handler is wrapped in the
	// Bearer middleware
	handler := mustNew(app.NewHTTPHandlerWithProducer(orderPool, txManager, inventoryClient, paymentClient, authSvcClient, realOrderProducer))
	ts = httptest.NewServer(handler)
	cleanups.add("httptest server", func(_ context.Context) error { ts.Close(); return nil })

	// 11. The Order ShipAssembled consumer — the real code from internal/consumer/assembly_consumer.
	// It listens on the ShipAssembled topic and moves the order to ASSEMBLED via CommitParts
	startOrderShipAssembledConsumer(ctx, cleanups, broker, orderPool, txManager, inventoryClient)

	// 12. The real AssemblyService — the same code that runs in production:
	// consumer/order_paid → service/assembly → producer/ship_assembled. The contract of
	// both proto messages is exercised through the real decode/encode logic of the
	// assembly service
	startAssemblyService(ctx, cleanups, broker)

	return m.Run()
}

// =============================================================================
// Containers & infrastructure helpers
// =============================================================================

func startPostgres(ctx context.Context, dbName, user, password string) (*tcpostgres.PostgresContainer, string, error) {
	c, err := tcpostgres.Run(
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

	dsn, err := c.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, "", err
	}

	return c, dsn, nil
}

func startPostgresAndMigrate(ctx context.Context, cleanups *cleanupStack, name, migrationsDir string) *pgxpool.Pool {
	container, dsn, err := startPostgres(ctx, name, name+"-user", name+"-password")
	if err != nil {
		panic(fmt.Errorf("postgres %s: %w", name, err))
	}
	cleanups.add("postgres "+name, func(c context.Context) error { return container.Terminate(c) })

	if err = runMigrations(dsn, migrationsDir); err != nil {
		panic(fmt.Errorf("migrate %s: %w", name, err))
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		panic(fmt.Errorf("pgxpool %s: %w", name, err))
	}
	cleanups.add("pgxpool "+name, func(_ context.Context) error { pool.Close(); return nil })

	return pool
}

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

// startRedis starts the Redis container for IAM sessions and returns a connected
// redis client; both the container and the client are closed through cleanups.
func startRedis(ctx context.Context, cleanups *cleanupStack) *redis.Client {
	container, err := tcredis.Run(ctx, redisImage)
	if err != nil {
		panic(fmt.Errorf("redis: %w", err))
	}
	cleanups.add("redis container", func(c context.Context) error { return container.Terminate(c) })

	addr, err := container.ConnectionString(ctx)
	if err != nil {
		panic(fmt.Errorf("redis connection string: %w", err))
	}

	const prefix = "redis://"
	if len(addr) > len(prefix) {
		addr = addr[len(prefix):]
	}

	client := redis.NewClient(&redis.Options{Addr: addr})
	cleanups.add("redis client", func(_ context.Context) error { return client.Close() })

	return client
}

func startRedpanda(ctx context.Context, cleanups *cleanupStack) string {
	container, err := tcredpanda.Run(
		ctx, redpandaImage,
		tcredpanda.WithAutoCreateTopics(),
	)
	if err != nil {
		panic(fmt.Errorf("redpanda: %w", err))
	}
	cleanups.add("redpanda", func(c context.Context) error { return container.Terminate(c) })

	broker, err := container.KafkaSeedBroker(ctx)
	if err != nil {
		panic(fmt.Errorf("redpanda broker addr: %w", err))
	}

	return broker
}

func createTopics(broker string, topics ...string) {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V2_8_0_0

	admin, err := sarama.NewClusterAdmin([]string{broker}, cfg)
	if err != nil {
		panic(fmt.Errorf("cluster admin: %w", err))
	}
	defer func() { _ = admin.Close() }()

	for _, topic := range topics {
		err = admin.CreateTopic(topic, &sarama.TopicDetail{
			NumPartitions:     topicPartitions,
			ReplicationFactor: topicReplicationFactor,
		}, false)
		if err != nil {
			panic(fmt.Errorf("create topic %q: %w", topic, err))
		}
	}
}

// startBufconnGRPCIAM starts the IAM gRPC server over bufconn and returns the client
// connection. bcrypt.MinCost is critical for test speed.
func startBufconnGRPCIAM(ctx context.Context, cleanups *cleanupStack, pool *pgxpool.Pool, redisClient *redis.Client) *grpc.ClientConn {
	lis := bufconn.Listen(bufSize)
	server := iamApp.NewGRPCServer(pool, redisClient, sessionTTL, bcrypt.MinCost)

	go func() {
		if err := server.Serve(lis); err != nil {
			panic(fmt.Errorf("iam grpc serve: %w", err))
		}
	}()
	cleanups.add("iam grpc server", func(_ context.Context) error { server.Stop(); return nil })

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(fmt.Errorf("iam grpc client: %w", err))
	}
	cleanups.add("iam grpc conn", func(_ context.Context) error { return conn.Close() })

	_ = ctx
	return conn
}

func startBufconnGRPCInventory(ctx context.Context, cleanups *cleanupStack, pool *pgxpool.Pool, authClient authv1.AuthServiceClient) *grpc.ClientConn {
	lis := bufconn.Listen(bufSize)
	server := grpc.NewServer(invApp.Interceptors(authClient)...)
	txManager := mustNew(manager.New(trmpgx.NewDefaultFactory(pool)))
	invApp.RegisterServices(txManager, server, pool)

	go func() {
		if err := server.Serve(lis); err != nil {
			panic(fmt.Errorf("inventory grpc serve: %w", err))
		}
	}()
	cleanups.add("inventory grpc server", func(_ context.Context) error { server.Stop(); return nil })

	// SessionForwarder copies the session-uuid from the context into the outgoing gRPC
	// metadata — needed both by assembly_consumer for CommitParts and by the getStock
	// helper in the test
	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(interceptor.SessionForwarder()),
	)
	if err != nil {
		panic(fmt.Errorf("inventory grpc client: %w", err))
	}
	cleanups.add("inventory grpc conn", func(_ context.Context) error { return conn.Close() })

	_ = ctx
	return conn
}

func startBufconnGRPCPayment(ctx context.Context, cleanups *cleanupStack) *grpc.ClientConn {
	lis := bufconn.Listen(bufSize)
	server := grpc.NewServer(payApp.Interceptors()...)
	payApp.RegisterServices(server)

	go func() {
		if err := server.Serve(lis); err != nil {
			panic(fmt.Errorf("payment grpc serve: %w", err))
		}
	}()
	cleanups.add("payment grpc server", func(_ context.Context) error { server.Stop(); return nil })

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(fmt.Errorf("payment grpc client: %w", err))
	}
	cleanups.add("payment grpc conn", func(_ context.Context) error { return conn.Close() })

	_ = ctx
	return conn
}

// =============================================================================
// Kafka consumers & test-assembler
// =============================================================================

func startOrderShipAssembledConsumer(
	ctx context.Context,
	cleanups *cleanupStack,
	broker string,
	pool *pgxpool.Pool,
	txManager *manager.Manager,
	invClient inventoryv1.InventoryServiceClient,
) {
	cg := mustNew(sarama.NewConsumerGroup([]string{broker}, orderGroupID, consumerConfig()))
	cleanups.add("order ship-assembled consumer group", func(_ context.Context) error { return cg.Close() })

	// The ConsumerSession middleware pulls session_uuid out of the Kafka header and puts
	// it into the context; without it assembly_consumer cannot call the protected
	// Inventory.CommitParts, which requires session-uuid in the gRPC metadata
	wrappedConsumer := wrappedKafkaConsumer.NewConsumer(
		cg,
		[]string{shipAssembledTopic},
		wrappedKafkaConsumer.WithMiddlewares(
			kafkaMiddleware.ConsumerSession(),
		),
	)

	// The real code from order/internal/consumer/assembly_consumer. The repository and
	// the inventory client come from the same internal packages the production DI uses
	// (order/internal/app/di.go)
	svc := assemblyconsumer.NewService(
		wrappedConsumer,
		orderRepoPkg.New(pool, txManager),
		inventoryClientPkg.New(invClient),
		txManager,
	)

	go func() {
		if err := svc.RunConsumer(ctx); err != nil {
			// On ctx cancellation Consume returns an error, which is expected — do not panic.
			// Log it for diagnostics in case the failure is real
			_, _ = fmt.Fprintf(os.Stderr, "order ship-assembled consumer stopped: %v\n", err)
		}
	}()
}

// startAssemblyService starts the real AssemblyService through assembly/pkg/app.
// It is the same code as in production: consumer/order_paid → service/assembly →
// producer/ship_assembled. The ConsumerSession middleware is wired inside pkg/app;
// without it session_uuid would not reach the ShipAssembled message.
//
// Since week 6 build_time is hardcoded to 5-15 seconds, so the ASSEMBLED wait timeout
// in the test is larger — see waitForOrderStatus in lifecycle_test.
func startAssemblyService(ctx context.Context, cleanups *cleanupStack, broker string) {
	cg := mustNew(sarama.NewConsumerGroup([]string{broker}, assemblyGroupID, consumerConfig()))
	cleanups.add("assembly consumer group", func(_ context.Context) error { return cg.Close() })

	syncProducer := mustNew(sarama.NewSyncProducer([]string{broker}, producerConfig()))
	cleanups.add("assembly sync producer", func(_ context.Context) error { return syncProducer.Close() })

	svc := assemblyApp.New(syncProducer, cg, assemblyApp.Config{
		OrderPaidTopic:     orderPaidTopic,
		ShipAssembledTopic: shipAssembledTopic,
	})

	go func() {
		if err := svc.RunConsumer(ctx); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "assembly service stopped: %v\n", err)
		}
	}()
}

func producerConfig() *sarama.Config {
	cfg := sarama.NewConfig()
	cfg.Producer.Return.Successes = true
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	return cfg
}

func consumerConfig() *sarama.Config {
	cfg := sarama.NewConfig()
	cfg.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRoundRobin()}
	cfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	return cfg
}

// =============================================================================
// Cleanup stack — LIFO shutdown order without a pile of defers in TestMain.
// =============================================================================

type cleanupStack struct {
	mu    sync.Mutex
	items []cleanupItem
}

type cleanupItem struct {
	name string
	fn   func(context.Context) error
}

func newCleanupStack() *cleanupStack {
	return &cleanupStack{}
}

func (s *cleanupStack) add(name string, fn func(context.Context) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = append(s.items, cleanupItem{name: name, fn: fn})
}

func (s *cleanupStack) run(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := len(s.items) - 1; i >= 0; i-- {
		item := s.items[i]
		if err := item.fn(ctx); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "cleanup %q failed: %v\n", item.name, err)
		}
	}
}

func mustNew[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
