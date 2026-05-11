package v1

import (
	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"

	errs "github.com/vixart/rocket-factory/order/internal/errors"
	"github.com/vixart/rocket-factory/order/internal/model"
	orderv1 "github.com/vixart/rocket-factory/shared/pkg/openapi/order/v1"
)

func (s *APISuite) TestPayOrderSuccess() {
	var (
		orderUUID = uuid.New()
		txUUID    = uuid.New()

		req = &orderv1.PayOrderRequest{
			PaymentMethod: orderv1.PaymentMethodCARD,
		}

		params = orderv1.PayOrderParams{
			OrderUUID: orderUUID,
		}

		expectedPaymentMethod = model.PaymentMethodCard
	)

	s.orderService.
		EXPECT().
		Pay(s.ctx, orderUUID, expectedPaymentMethod).
		Return(&txUUID, nil)

	res, err := s.api.PayOrder(s.ctx, req, params)

	s.Require().NoError(err)
	s.Require().NotNil(res)

	successRes, ok := res.(*orderv1.PayOrderResponse)

	s.Require().True(ok)
	s.Require().Equal(txUUID, successRes.TransactionUUID)
}

func (s *APISuite) TestPayOrderOrderNotFound() {
	var (
		orderUUID = uuid.New()

		req = &orderv1.PayOrderRequest{
			PaymentMethod: orderv1.PaymentMethodCARD,
		}

		params = orderv1.PayOrderParams{
			OrderUUID: orderUUID,
		}

		expectedPaymentMethod = model.PaymentMethodCard
	)

	s.orderService.
		EXPECT().
		Pay(s.ctx, orderUUID, expectedPaymentMethod).
		Return(nil, errs.ErrOrderNotFound)

	res, err := s.api.PayOrder(s.ctx, req, params)

	s.Require().NoError(err)
	s.Require().NotNil(res)

	notFoundRes, ok := res.(*orderv1.PayOrderNotFound)

	s.Require().True(ok)
	s.Require().Equal(404, notFoundRes.Code)
	s.Require().Equal("заказ не найден", notFoundRes.Message)
}

func (s *APISuite) TestPayOrderInvalidPaymentMethod() {
	var (
		orderUUID = uuid.New()

		req = &orderv1.PayOrderRequest{
			PaymentMethod: "WrongMethod",
		}

		params = orderv1.PayOrderParams{
			OrderUUID: orderUUID,
		}

		expectedPaymentMethod = model.PaymentMethodUnspecified
	)

	s.orderService.
		EXPECT().
		Pay(s.ctx, orderUUID, expectedPaymentMethod).
		Return(nil, errs.ErrInvalidPaymentMethod)

	res, err := s.api.PayOrder(s.ctx, req, params)

	s.Require().NoError(err)
	s.Require().NotNil(res)

	badRequestRes, ok := res.(*orderv1.PayOrderBadRequest)

	s.Require().True(ok)
	s.Require().Equal(400, badRequestRes.Code)
	s.Require().Equal("передан недопустимый метод оплаты", badRequestRes.Message)
}

func (s *APISuite) TestPayOrderInvalidOrderStatus() {
	var (
		orderUUID = uuid.New()

		req = &orderv1.PayOrderRequest{
			PaymentMethod: orderv1.PaymentMethodCARD,
		}

		params = orderv1.PayOrderParams{
			OrderUUID: orderUUID,
		}

		expectedPaymentMethod = model.PaymentMethodCard
	)

	s.orderService.
		EXPECT().
		Pay(s.ctx, orderUUID, expectedPaymentMethod).
		Return(nil, errs.ErrInvalidOrderStatus)

	res, err := s.api.PayOrder(s.ctx, req, params)

	s.Require().NoError(err)
	s.Require().NotNil(res)

	conflictRes, ok := res.(*orderv1.PayOrderConflict)

	s.Require().True(ok)
	s.Require().Equal(409, conflictRes.Code)
	s.Require().Equal("заказ имеет недопустимый статус", conflictRes.Message)
}

func (s *APISuite) TestPayOrderPaymentFailed() {
	var (
		orderUUID = uuid.New()

		req = &orderv1.PayOrderRequest{
			PaymentMethod: orderv1.PaymentMethodCARD,
		}

		params = orderv1.PayOrderParams{
			OrderUUID: orderUUID,
		}

		expectedPaymentMethod = model.PaymentMethodCard
	)

	s.orderService.
		EXPECT().
		Pay(s.ctx, orderUUID, expectedPaymentMethod).
		Return(nil, errs.ErrPaymentFailed)

	res, err := s.api.PayOrder(s.ctx, req, params)

	s.Require().NoError(err)
	s.Require().NotNil(res)

	internalRes, ok := res.(*orderv1.PayOrderInternalServerError)

	s.Require().True(ok)
	s.Require().Equal(500, internalRes.Code)
	s.Require().Equal("нет соединения с платежным сервисом", internalRes.Message)
}

func (s *APISuite) TestPayOrderUnknownError() {
	var (
		orderUUID  = uuid.New()
		unknownErr = gofakeit.Error()

		req = &orderv1.PayOrderRequest{
			PaymentMethod: orderv1.PaymentMethodCARD,
		}

		params = orderv1.PayOrderParams{
			OrderUUID: orderUUID,
		}

		expectedPaymentMethod = model.PaymentMethodCard
	)

	s.orderService.
		EXPECT().
		Pay(s.ctx, orderUUID, expectedPaymentMethod).
		Return(nil, unknownErr)

	res, err := s.api.PayOrder(s.ctx, req, params)

	s.Require().NoError(err)
	s.Require().NotNil(res)

	internalRes, ok := res.(*orderv1.PayOrderInternalServerError)

	s.Require().True(ok)
	s.Require().Equal(500, internalRes.Code)
	s.Require().Equal("что-то пошло не так", internalRes.Message)
}
