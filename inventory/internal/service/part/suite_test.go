package part

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/vixart/rocket-factory/inventory/internal/service/part/mocks"
)

type ServiceSuite struct {
	suite.Suite

	ctx context.Context

	partRepo *mocks.Repository

	service *service
}

func (s *ServiceSuite) SetupTest() {
	s.ctx = context.Background()

	s.partRepo = mocks.NewRepository(s.T())

	s.service = NewService(s.partRepo)
}

func (s *ServiceSuite) TearDownTest() {
	s.T().Log("TearDownTest: очистка после", s.T().Name())
}

func TestServiceSuite(t *testing.T) {
	suite.Run(t, new(ServiceSuite))
}
