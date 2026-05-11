package payment

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ServiceSuite struct {
	suite.Suite

	ctx context.Context

	service *service
}

func (s *ServiceSuite) SetupTest() {
	s.ctx = context.Background()

	s.service = NewService()
}

func (s *ServiceSuite) TearDownTest() {
	s.T().Log("TearDownTest: очистка после", s.T().Name())
}

func TestServiceSuite(t *testing.T) {
	suite.Run(t, new(ServiceSuite))
}
