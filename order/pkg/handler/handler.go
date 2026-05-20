package handler

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	orderv1 "github.com/vixart/rocket-factory/shared/pkg/openapi/order/v1"
	inventoryv1 "github.com/vixart/rocket-factory/shared/pkg/proto/inventory/v1"
	paymentv1 "github.com/vixart/rocket-factory/shared/pkg/proto/payment/v1"
)

// Order представляет заказ на постройку космического корабля.
type Order struct {
	OrderUUID       uuid.UUID
	HullUUID        uuid.UUID
	EngineUUID      uuid.UUID
	ShieldUUID      *uuid.UUID // опциональный
	WeaponUUID      *uuid.UUID // опциональный
	TotalPrice      int64      // в копейках
	TransactionUUID *uuid.UUID
	PaymentMethod   *string
	Status          string // PENDING_PAYMENT, PAID, CANCELLED
	CreatedAt       time.Time
}

// OrderStore — хранилище заказов (in-memory).
type OrderStore struct {
	mu     sync.RWMutex
	orders map[uuid.UUID]Order
}

// NewOrderStore создаёт новое пустое хранилище заказов.
func NewOrderStore() *OrderStore {
	return &OrderStore{
		orders: make(map[uuid.UUID]Order),
	}
}

// OrderHandler реализует интерфейс orderv1.Handler, сгенерированный ogen.
type OrderHandler struct {
	orderv1.UnimplementedHandler
	inventoryClient inventoryv1.InventoryServiceClient
	paymentClient   paymentv1.PaymentServiceClient
	store           *OrderStore
}

// NewOrderHandler создаёт новый обработчик заказов.
func NewOrderHandler(
	inventoryClient inventoryv1.InventoryServiceClient,
	paymentClient paymentv1.PaymentServiceClient,
	store *OrderStore,
) *OrderHandler {
	return &OrderHandler{
		inventoryClient: inventoryClient,
		paymentClient:   paymentClient,
		store:           store,
	}
}

// SetupServer создаёт OpenAPI сервер на основе обработчика.
func SetupServer(h *OrderHandler) (*orderv1.Server, error) {
	return orderv1.NewServer(h)
}

// GetOrder реализует операцию getOrder (пример реализации).
// GET /api/v1/orders/{order_uuid}.
func (h *OrderHandler) GetOrder(_ context.Context, params orderv1.GetOrderParams) (orderv1.GetOrderRes, error) {
	// 1. Найти заказ в store (с блокировкой для thread-safety).
	h.store.mu.RLock()
	order, ok := h.store.orders[params.OrderUUID]
	h.store.mu.RUnlock()

	// 2. Если не найден — вернуть 404.
	if !ok {
		return &orderv1.GetOrderNotFound{
			Code:    http.StatusNotFound,
			Message: "заказ не найден",
		}, nil
	}

	// 3. Преобразовать в DTO и вернуть.
	var shieldUUID orderv1.OptNilUUID
	if order.ShieldUUID != nil {
		shieldUUID = orderv1.NewOptNilUUID(*order.ShieldUUID)
	}

	var weaponUUID orderv1.OptNilUUID
	if order.WeaponUUID != nil {
		weaponUUID = orderv1.NewOptNilUUID(*order.WeaponUUID)
	}

	var transactionUUID orderv1.OptNilUUID
	if order.TransactionUUID != nil {
		transactionUUID = orderv1.NewOptNilUUID(*order.TransactionUUID)
	}

	var paymentMethod orderv1.OptNilPaymentMethod
	if order.PaymentMethod != nil {
		paymentMethod = orderv1.NewOptNilPaymentMethod(orderv1.PaymentMethod(*order.PaymentMethod))
	}

	return &orderv1.OrderDto{
		OrderUUID:       order.OrderUUID,
		HullUUID:        order.HullUUID,
		EngineUUID:      order.EngineUUID,
		ShieldUUID:      shieldUUID,
		WeaponUUID:      weaponUUID,
		TotalPrice:      order.TotalPrice,
		TransactionUUID: transactionUUID,
		PaymentMethod:   paymentMethod,
		Status:          orderv1.OrderStatus(order.Status),
		CreatedAt:       order.CreatedAt,
	}, nil
}

// CreateOrder реализует операцию createOrder.
// POST /api/v1/orders.
func (h *OrderHandler) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (orderv1.CreateOrderRes, error) {
	engineUUID := req.GetEngineUUID().String()
	hullUUID := req.GetHullUUID().String()

	uuids := []string{engineUUID, hullUUID}

	var shieldUUID, weaponUUID string

	if v, ok := req.GetShieldUUID().Get(); ok {
		shieldUUID = v.String()
		uuids = append(uuids, shieldUUID)
	}

	if v, ok := req.GetWeaponUUID().Get(); ok {
		weaponUUID = v.String()
		uuids = append(uuids, weaponUUID)
	}

	parts, err := h.listParts(ctx, uuids)
	if err != nil {
		return mapCreateOrderError(err), nil
	}

	partsByUUID := make(map[string]*inventoryv1.Part, len(parts))
	for _, p := range parts {
		partsByUUID[p.GetUuid()] = p
	}

	notFound := func(msg string) orderv1.CreateOrderRes {
		return &orderv1.CreateOrderBadRequest{
			Message: msg,
			Code:    http.StatusNotFound,
		}
	}

	// обязательные части
	enginePart, ok := partsByUUID[engineUUID]
	if !ok {
		return notFound("не удалось найти двигатель"), nil
	}

	hullPart, ok := partsByUUID[hullUUID]
	if !ok {
		return notFound("не удалось найти корпус"), nil
	}

	// опциональные
	var shieldPart, weaponPart *inventoryv1.Part

	if shieldUUID != "" {
		shieldPart, ok = partsByUUID[shieldUUID]
		if !ok {
			return notFound("не удалось найти щит"), nil
		}
	}

	if weaponUUID != "" {
		weaponPart, ok = partsByUUID[weaponUUID]
		if !ok {
			return notFound("не удалось найти оружие"), nil
		}
	}

	totalPrice := enginePart.GetPrice() + hullPart.GetPrice()
	if shieldPart != nil {
		totalPrice += shieldPart.Price
	}
	if weaponPart != nil {
		totalPrice += weaponPart.Price
	}

	orderUUID := uuid.New()
	order := Order{
		OrderUUID:  orderUUID,
		HullUUID:   req.GetHullUUID(),
		EngineUUID: req.GetEngineUUID(),
		TotalPrice: totalPrice,
		Status:     string(orderv1.OrderStatusPENDINGPAYMENT),
		CreatedAt:  time.Now(),
	}
	if shieldPart != nil {
		order.ShieldUUID = new(uuid.MustParse(shieldPart.GetUuid()))
	}
	if weaponPart != nil {
		order.WeaponUUID = new(uuid.MustParse(weaponPart.GetUuid()))
	}

	h.store.mu.Lock()
	defer h.store.mu.Unlock()

	h.store.orders[orderUUID] = order

	return &orderv1.CreateOrderResponse{
		OrderUUID:  orderUUID,
		TotalPrice: totalPrice,
	}, nil
}

func (h *OrderHandler) listParts(ctx context.Context, uuids []string) (map[string]*inventoryv1.Part, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := h.inventoryClient.ListParts(ctxWithTimeout, &inventoryv1.ListPartsRequest{
		Uuids: uuids,
	})
	if err != nil {
		return nil, err
	}

	parts := resp.GetParts()

	result := make(map[string]*inventoryv1.Part, len(parts))
	for _, p := range parts {
		if p.GetStockQuantity() <= 0 {
			return nil, status.Errorf(
				codes.FailedPrecondition,
				"детали нет на складе: %s - %s",
				p.GetName(),
				p.GetUuid(),
			)
		}
		result[p.GetUuid()] = p
	}

	return result, nil
}

func mapCreateOrderError(err error) orderv1.CreateOrderRes {
	st, ok := status.FromError(err)
	if !ok {
		return &orderv1.CreateOrderInternalServerError{
			Code:    http.StatusInternalServerError,
			Message: "непоправимая ошибка",
		}
	}

	switch st.Code() {
	case codes.NotFound:
		return &orderv1.CreateOrderNotFound{
			Code:    http.StatusNotFound,
			Message: st.Message(),
		}

	case codes.InvalidArgument:
		return &orderv1.CreateOrderBadRequest{
			Code:    http.StatusBadRequest,
			Message: st.Message(),
		}

	case codes.FailedPrecondition:
		return &orderv1.CreateOrderConflict{
			Code:    http.StatusConflict,
			Message: st.Message(),
		}

	default:
		return &orderv1.CreateOrderInternalServerError{
			Code:    http.StatusInternalServerError,
			Message: st.Message(),
		}
	}
}

func mapPayOrderError(err error) orderv1.PayOrderRes {
	st, ok := status.FromError(err)
	if !ok {
		return &orderv1.PayOrderInternalServerError{
			Code:    http.StatusInternalServerError,
			Message: "непоправимая ошибка",
		}
	}

	switch st.Code() {
	case codes.InvalidArgument:
		return &orderv1.PayOrderBadRequest{
			Code:    http.StatusBadRequest,
			Message: st.Message(),
		}
	default:
		return &orderv1.PayOrderInternalServerError{
			Code:    http.StatusInternalServerError,
			Message: st.Message(),
		}
	}
}

// PayOrder реализует операцию payOrder.
// POST /api/v1/orders/{order_uuid}/pay.
func (h *OrderHandler) PayOrder(ctx context.Context, req *orderv1.PayOrderRequest, params orderv1.PayOrderParams) (orderv1.PayOrderRes, error) {
	// 1. Найти заказ в store (с блокировкой для thread-safety).
	h.store.mu.RLock()
	order, ok := h.store.orders[params.OrderUUID]
	h.store.mu.RUnlock()

	if !ok {
		return &orderv1.PayOrderNotFound{
			Code:    http.StatusNotFound,
			Message: "заказ не найден",
		}, nil
	}

	if order.Status != string(orderv1.OrderStatusPENDINGPAYMENT) {
		return &orderv1.PayOrderConflict{
			Code:    http.StatusConflict,
			Message: "неверный статус заказа",
		}, nil
	}

	payOrderReq := paymentv1.PayOrderRequest{
		OrderUuid:     order.OrderUUID.String(),
		PaymentMethod: mapPaymentMethod(req.PaymentMethod),
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	payOrderRes, err := h.paymentClient.PayOrder(ctxWithTimeout, &payOrderReq)
	if err != nil {
		return mapPayOrderError(err), nil
	}

	order.Status = string(orderv1.OrderStatusPAID)
	order.TransactionUUID = new(uuid.MustParse(payOrderRes.TransactionUuid))
	order.PaymentMethod = new(string(req.PaymentMethod))

	h.store.mu.Lock()
	h.store.orders[order.OrderUUID] = order
	h.store.mu.Unlock()

	return &orderv1.PayOrderResponse{
		TransactionUUID: *order.TransactionUUID,
	}, nil
}

func mapPaymentMethod(m orderv1.PaymentMethod) paymentv1.PaymentMethod {
	switch m {
	case orderv1.PaymentMethodCARD:
		return paymentv1.PaymentMethod_PAYMENT_METHOD_CARD
	case orderv1.PaymentMethodSBP:
		return paymentv1.PaymentMethod_PAYMENT_METHOD_SBP
	case orderv1.PaymentMethodCREDITCARD:
		return paymentv1.PaymentMethod_PAYMENT_METHOD_CREDIT_CARD
	case orderv1.PaymentMethodINVESTORMONEY:
		return paymentv1.PaymentMethod_PAYMENT_METHOD_INVESTOR_MONEY
	default:
		return paymentv1.PaymentMethod_PAYMENT_METHOD_UNSPECIFIED
	}
}

// CancelOrder реализует операцию cancelOrder.
// POST /api/v1/orders/{order_uuid}/cancel.
func (h *OrderHandler) CancelOrder(ctx context.Context, params orderv1.CancelOrderParams) (orderv1.CancelOrderRes, error) {
	h.store.mu.RLock()
	order, ok := h.store.orders[params.OrderUUID]
	h.store.mu.RUnlock()

	if !ok {
		return &orderv1.CancelOrderNotFound{
			Code:    http.StatusNotFound,
			Message: "заказ не найден",
		}, nil
	}

	if order.Status != string(orderv1.OrderStatusPENDINGPAYMENT) {
		return &orderv1.CancelOrderConflict{
			Code:    http.StatusConflict,
			Message: "неверный статус заказа",
		}, nil
	}

	order.Status = string(orderv1.OrderStatusCANCELLED)

	h.store.mu.Lock()
	h.store.orders[order.OrderUUID] = order
	h.store.mu.Unlock()

	return &orderv1.CancelOrderResponse{}, nil
}
