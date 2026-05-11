package payment

import (
	"github.com/google/uuid"

	errs "github.com/vixart/rocket-factory/payment/internal/errors"
	"github.com/vixart/rocket-factory/payment/internal/model"
)

func (s *ServiceSuite) TestPayOrderSuccess() {
	var (
		orderUUID     = uuid.New()
		paymentMethod = model.PaymentMethodCard
	)

	txUUID, err := s.service.PayOrder(
		s.ctx,
		orderUUID,
		paymentMethod,
	)

	s.Require().NoError(err)
	s.Require().NotNil(txUUID)
	s.Require().NotEqual(uuid.Nil, *txUUID)
}

func (s *ServiceSuite) TestPayOrderUnspecifiedMethod() {
	var (
		orderUUID     = uuid.New()
		paymentMethod = model.PaymentMethodUnspecified
	)

	txUUID, err := s.service.PayOrder(
		s.ctx,
		orderUUID,
		paymentMethod,
	)

	s.Require().Error(err)
	s.Require().Nil(txUUID)

	s.Require().ErrorIs(err, errs.ErrPaymentMethodNotSpecified)
}
