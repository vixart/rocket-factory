//go:build apitest

package tests

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	inventoryv1 "github.com/vixart/rocket-factory/shared/pkg/proto/inventory/v1"
)

// The concurrency tests come from week_4 (where they flaked because SELECT FOR UPDATE
// was missing) and week_5 (where FOR UPDATE was introduced). They still hold: the
// pessimistic locks in Inventory remain — ListForUpdate takes a row lock on read, so
// the second call waits and observes the updated state.
//
//
// The direct gRPC scenario for ReserveParts is covered by
// TestInventory_ReserveParts_Concurrent_LastPart in api_test.go; the tests below add the
// HTTP chain through Order and the transaction atomicity case with mixed stock.

// TestConcurrent_CreateOrder_LastUnit_ExactlyOneSucceeds: two goroutines create an
// order for the same part with stock=1 at the same time. FOR UPDATE in
// Inventory.ReserveParts must let exactly one through; the other gets 409
// (inventory's ErrOutOfStock maps to HTTP 409 in the order error handler).
// error_handler.go).
func TestConcurrent_CreateOrder_LastUnit_ExactlyOneSucceeds(t *testing.T) {
	hullUUID := uuid.New().String()
	engineUUID := uuid.New().String()
	_, err := inventoryDBPool.Exec(
		context.Background(),
		`INSERT INTO parts (uuid, name, description, part_type, price, stock_quantity, properties)
		 VALUES
		   ($1, 'Concurrent last unit hull', '', 'HULL', 1000, 1, '{"hull": {"strength": 100}}'),
		   ($2, 'Concurrent last unit engine', '', 'ENGINE', 1000, 5, '{"engine": {"class": "C", "required_strength": 50}}')`,
		hullUUID, engineUUID,
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = inventoryDBPool.Exec(context.Background(),
			`DELETE FROM parts WHERE uuid IN ($1, $2)`, hullUUID, engineUUID)
	})

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		statuses []int
	)
	wg.Add(2)
	for range 2 {
		go func() {
			defer wg.Done()
			_, resp := createOrder(t, &CreateOrderRequest{
				UserUUID:   uuid.New().String(),
				HullUUID:   hullUUID,
				EngineUUID: engineUUID,
			})
			defer func() { _ = resp.Body.Close() }()
			mu.Lock()
			defer mu.Unlock()
			statuses = append(statuses, resp.StatusCode)
		}()
	}
	wg.Wait()

	created := 0
	conflict := 0
	for _, s := range statuses {
		switch s {
		case http.StatusCreated:
			created++
		case http.StatusConflict:
			conflict++
		}
	}
	assert.Equal(t, 1, created, "exactly one order must be created (statuses=%v)", statuses)
	assert.Equal(t, 1, conflict, "the second order must get Conflict (statuses=%v)", statuses)
}

// TestConcurrent_Reserve_MixedStock: concurrent ReserveParts with a batch of two parts
// where one is available and the other is out of stock (HullOutOfStockUUID, stock=0 in
// the seed). The point is transaction integrity in ReserveParts: if a single part fails,
// no reservation is persisted at all.
func TestConcurrent_Reserve_MixedStock(t *testing.T) {
	availableUUID := uuid.New().String()
	_, err := inventoryDBPool.Exec(
		context.Background(),
		`INSERT INTO parts (uuid, name, description, part_type, price, stock_quantity, properties)
		 VALUES ($1, 'Concurrent mixed available', '', 'HULL', 1000, 5, '{"hull": {"strength": 100}}')`,
		availableUUID,
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = inventoryDBPool.Exec(context.Background(),
			`DELETE FROM parts WHERE uuid = $1`, availableUUID)
	})

	const workers = 4

	var (
		wg        sync.WaitGroup
		exhausted atomic.Int64
		others    atomic.Int64
	)

	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(authCtx(context.Background()), 10*time.Second)
			defer cancel()
			_, err := inventoryClient.ReserveParts(ctx,
				&inventoryv1.ReservePartsRequest{
					Uuids: []string{availableUUID, HullOutOfStockUUID},
				})
			require.Error(t, err)
			// Inventory maps ErrOutOfStock to ResourceExhausted (interceptor/error.go).
			if status.Code(err) == codes.ResourceExhausted {
				exhausted.Add(1)
			} else {
				others.Add(1)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int64(0), others.Load(),
		"every call must fail with ResourceExhausted and nothing else")
	assert.Equal(t, int64(workers), exhausted.Load(),
		"every batch must fail as a whole because of HullOutOfStockUUID")

	// The key assertion: no available part may stay reserved, because the transaction
	// rolled back. The reserved field is not in the proto (it is an Inventory internal),
	// so it is read directly from the database.
	var reserved int
	err = inventoryDBPool.QueryRow(context.Background(),
		`SELECT reserved FROM parts WHERE uuid = $1`, availableUUID).Scan(&reserved)
	require.NoError(t, err)
	assert.Equal(t, 0, reserved,
		"availableUUID must not stay reserved: the transaction rolled back")
}
