//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vixart/rocket-factory/platform/pkg/auth"
	authv1 "github.com/vixart/rocket-factory/shared/pkg/proto/auth/v1"
	commonv1 "github.com/vixart/rocket-factory/shared/pkg/proto/common/v1"
	inventoryv1 "github.com/vixart/rocket-factory/shared/pkg/proto/inventory/v1"
	userv1 "github.com/vixart/rocket-factory/shared/pkg/proto/user/v1"
)

// The HTTP DTOs are duplicated from api_test on purpose: e2e is a standalone suite
// that keeps its contract with the HTTP API explicit.
// Since week 6 user_uuid is no longer part of the request body: OrderService takes it
// from the session (the middleware puts it into ctx).

type createOrderRequest struct {
	HullUUID   string  `json:"hull_uuid"`
	EngineUUID string  `json:"engine_uuid"`
	ShieldUUID *string `json:"shield_uuid,omitempty"`
	WeaponUUID *string `json:"weapon_uuid,omitempty"`
}

type createOrderResponse struct {
	OrderUUID  string `json:"order_uuid"`
	TotalPrice int64  `json:"total_price"`
}

type payOrderRequest struct {
	PaymentMethod string `json:"payment_method"`
}

type payOrderResponse struct {
	TransactionUUID string `json:"transaction_uuid"`
}

type orderDTO struct {
	OrderUUID       string  `json:"order_uuid"`
	UserUUID        string  `json:"user_uuid"`
	HullUUID        string  `json:"hull_uuid"`
	EngineUUID      string  `json:"engine_uuid"`
	TotalPrice      int64   `json:"total_price"`
	TransactionUUID *string `json:"transaction_uuid"`
	PaymentMethod   *string `json:"payment_method"`
	Status          string  `json:"status"`
}

// TestE2E_OrderFullLifecycle_Assembled is the happy path through the WHOLE Kafka chain.
//
// Steps:
//  1. Register and log in to IAM to obtain the sessionUUID for the Bearer token
//  2. POST /orders — create the order (with Authorization: Bearer), expect 201
//  3. POST /orders/{uuid}/pay — pay for it, expect 200 and status PAID
//  4. order produces OrderPaid → the real AssemblyService → ShipAssembled
//  5. the order ship-assembled consumer handles the event and moves it to ASSEMBLED
//  6. Eventually GET /orders/{uuid} — status ASSEMBLED, transaction_uuid persisted
//  7. Check that CommitParts really decremented stock_quantity
func TestE2E_OrderFullLifecycle_Assembled(t *testing.T) {
	ctx := context.Background()

	// 1. Register and log in: one user for the whole test
	sessionUUID := registerAndLogin(t, ctx)

	// Snapshot of stock_quantity BEFORE the order, to verify CommitParts later.
	// session_uuid travels through ctx: the bufconn Inventory client carries a
	// SessionForwarder that puts it into the outgoing gRPC metadata.
	sessionUUIDParsed, err := uuid.Parse(sessionUUID)
	require.NoError(t, err)
	authCtx := auth.WithSessionUUID(ctx, sessionUUIDParsed)
	stockBefore := getStock(authCtx, t, []string{HullAluminumUUID, EngineIonCUUID})

	// 2. Create
	order := mustCreateOrder(t, sessionUUID, &createOrderRequest{
		HullUUID:   HullAluminumUUID,
		EngineUUID: EngineIonCUUID,
	})
	require.Equal(t, int64(HullAluminumPrice+EngineIonCPrice), order.TotalPrice)

	// Right after Create the status is PENDING_PAYMENT
	got := mustGetOrder(t, sessionUUID, order.OrderUUID)
	require.Equal(t, "PENDING_PAYMENT", got.Status)
	require.Nil(t, got.TransactionUUID)

	// 3. Pay
	pay := mustPayOrder(t, sessionUUID, order.OrderUUID, &payOrderRequest{PaymentMethod: "CARD"})
	require.NotEmpty(t, pay.TransactionUUID)

	// Right after Pay the status is PAID; ASSEMBLED comes later, the chain is asynchronous
	got = mustGetOrder(t, sessionUUID, order.OrderUUID)
	require.Equal(t, "PAID", got.Status)
	require.NotNil(t, got.TransactionUUID)
	assert.Equal(t, pay.TransactionUUID, *got.TransactionUUID)

	// 4-6. Wait for ASSEMBLED. Generous timeout: assembly simulates a 5-15 second build
	// (hardcoded in week 6), plus consumer group rebalancing and the message round trip
	// through Redpanda. 60 seconds is a comfortable margin.
	waitForOrderStatus(t, sessionUUID, order.OrderUUID, "ASSEMBLED", 60*time.Second)

	// Final check: every key field is persisted and the chain completed
	final := mustGetOrder(t, sessionUUID, order.OrderUUID)
	assert.Equal(t, "ASSEMBLED", final.Status)
	require.NotNil(t, final.TransactionUUID)
	assert.Equal(t, pay.TransactionUUID, *final.TransactionUUID)
	require.NotNil(t, final.PaymentMethod)
	assert.Equal(t, "CARD", *final.PaymentMethod)

	// 7. CommitParts must have decremented each used part by 1. This is the contract of
	// ShipAssembledHandler → InventoryClient.CommitParts, which api_test cannot cover
	// because it uses a noopProducer.
	stockAfter := getStock(authCtx, t, []string{HullAluminumUUID, EngineIonCUUID})
	assert.Equal(t, stockBefore[HullAluminumUUID]-1, stockAfter[HullAluminumUUID],
		"hull stock must drop by 1 after ASSEMBLED")
	assert.Equal(t, stockBefore[EngineIonCUUID]-1, stockAfter[EngineIonCUUID],
		"engine stock must drop by 1 after ASSEMBLED")
}

// =============================================================================
// IAM helper
// =============================================================================

// registerAndLogin registers a unique user and logs in immediately.
// It returns the sessionUUID, used both in Authorization: Bearer for HTTP requests
// and in auth.WithSessionUUID for direct gRPC calls to Inventory.
func registerAndLogin(t *testing.T, ctx context.Context) string {
	t.Helper()

	login := "e2e-" + uuid.New().String()[:8]
	const password = "password123"

	_, err := userSvcClient.Register(ctx, &userv1.RegisterRequest{
		Info: &userv1.UserRegistrationInfo{
			Info:     &commonv1.UserInfo{Login: login},
			Password: password,
		},
	})
	require.NoError(t, err, "register")

	loginResp, err := authSvcClient.Login(ctx, &authv1.LoginRequest{
		Login:    login,
		Password: password,
	})
	require.NoError(t, err, "login")

	sessionUUID := loginResp.GetSessionUuid()
	require.NotEmpty(t, sessionUUID, "session uuid")

	return sessionUUID
}

// =============================================================================
// HTTP helpers
// =============================================================================

func mustCreateOrder(t *testing.T, sessionUUID string, req *createOrderRequest) *createOrderResponse {
	t.Helper()

	body, err := json.Marshal(req)
	require.NoError(t, err)

	httpReq, err := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/orders", bytes.NewReader(body))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+sessionUUID)

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var out createOrderResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	return &out
}

func mustPayOrder(t *testing.T, sessionUUID, orderUUID string, req *payOrderRequest) *payOrderResponse {
	t.Helper()

	body, err := json.Marshal(req)
	require.NoError(t, err)

	httpReq, err := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/orders/"+orderUUID+"/pay", bytes.NewReader(body))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+sessionUUID)

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var out payOrderResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	return &out
}

func mustGetOrder(t *testing.T, sessionUUID, orderUUID string) *orderDTO {
	t.Helper()

	httpReq, err := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/orders/"+orderUUID, nil)
	require.NoError(t, err)
	httpReq.Header.Set("Authorization", "Bearer "+sessionUUID)

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var out orderDTO
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	return &out
}

// waitForOrderStatus polls GET /orders/{uuid} until the status matches expected.
// require.Eventually is the idiomatic waiting pattern in Go tests: it states
// "wait for a final state" rather than "sleep for a fixed interval".
// 200ms between attempts is a compromise: more often adds load, less often risks
// missing the window between the status change and the test pause.
func waitForOrderStatus(t *testing.T, sessionUUID, orderUUID, expected string, timeout time.Duration) {
	t.Helper()

	var lastStatus string
	require.Eventuallyf(t, func() bool {
		got := mustGetOrder(t, sessionUUID, orderUUID)
		lastStatus = got.Status
		return got.Status == expected
	}, timeout, 200*time.Millisecond,
		"order did not reach expected status: order_uuid=%s expected=%s last_seen=%s",
		orderUUID, expected, lastStatus)
}

// =============================================================================
// Inventory helper
// =============================================================================

// getStock reads part stock_quantity directly over bufconn gRPC.
// The caller must put session_uuid into ctx beforehand (via auth.WithSessionUUID):
// SessionForwarder then copies it into the outgoing gRPC metadata automatically.
func getStock(ctx context.Context, t *testing.T, partUUIDs []string) map[string]int64 {
	t.Helper()

	stocks := make(map[string]int64, len(partUUIDs))
	for _, u := range partUUIDs {
		resp, err := inventoryClient.GetPart(ctx, &inventoryv1.GetPartRequest{Uuid: u})
		require.NoError(t, err, "GetPart %s", u)
		stocks[u] = resp.GetPart().GetStockQuantity()
	}
	return stocks
}
