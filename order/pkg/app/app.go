package app

import (
	"context"
	"fmt"
	"net/http"

	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"github.com/jackc/pgx/v5/pgxpool"

	OrderApiV1 "github.com/vixart/rocket-factory/order/internal/api/order/v1"
	iamClientV1 "github.com/vixart/rocket-factory/order/internal/client/grpc/iam/v1"
	inventoryClientV1 "github.com/vixart/rocket-factory/order/internal/client/grpc/inventory/v1"
	paymentClientV1 "github.com/vixart/rocket-factory/order/internal/client/grpc/payment/v1"
	"github.com/vixart/rocket-factory/order/internal/middleware"
	"github.com/vixart/rocket-factory/order/internal/model"
	orderRepository "github.com/vixart/rocket-factory/order/internal/repository/order"
	orderService "github.com/vixart/rocket-factory/order/internal/service/order"
	orderv1 "github.com/vixart/rocket-factory/shared/pkg/openapi/order/v1"
	authv1 "github.com/vixart/rocket-factory/shared/pkg/proto/auth/v1"
	inventoryv1 "github.com/vixart/rocket-factory/shared/pkg/proto/inventory/v1"
	paymentv1 "github.com/vixart/rocket-factory/shared/pkg/proto/payment/v1"
)

func NewHTTPHandler(
	pool *pgxpool.Pool,
	txManager *manager.Manager,
	inventoryServiceClient inventoryv1.InventoryServiceClient,
	paymentServiceClient paymentv1.PaymentServiceClient,
	authServiceClient authv1.AuthServiceClient,
) (http.Handler, error) {
	inventoryClient := inventoryClientV1.New(inventoryServiceClient)
	paymentClient := paymentClientV1.NewClient(paymentServiceClient)
	iamClient := iamClientV1.NewClient(authServiceClient)
	orderRepo := orderRepository.New(pool, txManager)
	service := orderService.NewService(orderRepo, NewOrderPaidProducerServiceStub(), inventoryClient, paymentClient, txManager)
	api := OrderApiV1.NewApi(service)

	authMiddleware := middleware.NewAuthMiddleware(iamClient)
	orderServer, err := orderv1.NewServer(api, orderv1.WithErrorHandler(OrderApiV1.ErrorHandler))
	if err != nil {
		return nil, fmt.Errorf("ошибка создания сервера OpenAPI: %w", err)
	}
	return authMiddleware.AuthMiddleware(orderServer), nil
}

func NewHTTPHandlerWithProducer(
	pool *pgxpool.Pool,
	txManager *manager.Manager,
	inventoryServiceClient inventoryv1.InventoryServiceClient,
	paymentServiceClient paymentv1.PaymentServiceClient,
	authServiceClient authv1.AuthServiceClient,
	orderProducer orderService.OrderPaidProducer,
) (http.Handler, error) {
	inventoryClient := inventoryClientV1.New(inventoryServiceClient)
	paymentClient := paymentClientV1.NewClient(paymentServiceClient)
	iamClient := iamClientV1.NewClient(authServiceClient)
	orderRepo := orderRepository.New(pool, txManager)
	service := orderService.NewService(orderRepo, orderProducer, inventoryClient, paymentClient, txManager)
	api := OrderApiV1.NewApi(service)

	authMiddleware := middleware.NewAuthMiddleware(iamClient)
	orderServer, err := orderv1.NewServer(api, orderv1.WithErrorHandler(OrderApiV1.ErrorHandler))
	if err != nil {
		return nil, fmt.Errorf("ошибка создания сервера OpenAPI: %w", err)
	}
	return authMiddleware.AuthMiddleware(orderServer), nil
}

type OrderPaidProducerServiceStub struct{}

func NewOrderPaidProducerServiceStub() *OrderPaidProducerServiceStub {
	return &OrderPaidProducerServiceStub{}
}

func (o *OrderPaidProducerServiceStub) ProduceOrderPaid(
	_ context.Context,
	_ model.OrderPaidEvent,
) error {
	return nil
}
