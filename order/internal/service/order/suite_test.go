package order

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/vixart/rocket-factory/order/internal/service/order/mocks"
)

type ServiceSuite struct {
	suite.Suite

	ctx context.Context

	orderRepo       *mocks.Repository
	inventoryClient *mocks.InventoryClient
	paymentClient   *mocks.PaymentClient

	service *service
}

func (s *ServiceSuite) SetupTest() {
	s.ctx = context.Background()

	s.orderRepo = mocks.NewRepository(s.T())
	s.inventoryClient = mocks.NewInventoryClient(s.T())
	s.paymentClient = mocks.NewPaymentClient(s.T())

	s.service = NewService(s.orderRepo, s.inventoryClient, s.paymentClient)
}

func (s *ServiceSuite) TearDownTest() {
	s.T().Log("TearDownTest: очистка после", s.T().Name())
}

func TestServiceSuite(t *testing.T) {
	suite.Run(t, new(ServiceSuite))
}
