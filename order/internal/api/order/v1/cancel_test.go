package v1

import (
	"errors"
	"net/http"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"

	errs "github.com/vixart/rocket-factory/order/internal/errors"
	orderv1 "github.com/vixart/rocket-factory/shared/pkg/openapi/order/v1"
)

func (s *APISuite) TestCancelOrderSuccess() {
	var (
		orderUUID = uuid.New()

		params = orderv1.CancelOrderParams{
			OrderUUID: orderUUID,
		}
	)

	s.orderService.
		EXPECT().
		Cancel(s.ctx, orderUUID).
		Return(nil)

	res, err := s.api.CancelOrder(s.ctx, params)

	s.Require().NoError(err)
	s.Require().NotNil(res)

	response, ok := res.(*orderv1.CancelOrderResponse)

	s.Require().True(ok)
	s.Require().NotNil(response)
}

func (s *APISuite) TestCancelOrderNotFound() {
	var (
		orderUUID = uuid.New()

		params = orderv1.CancelOrderParams{
			OrderUUID: orderUUID,
		}
	)

	s.orderService.
		EXPECT().
		Cancel(s.ctx, orderUUID).
		Return(errs.ErrOrderNotFound)

	res, err := s.api.CancelOrder(s.ctx, params)

	s.Require().NoError(err)
	s.Require().NotNil(res)

	response, ok := res.(*orderv1.CancelOrderNotFound)

	s.Require().True(ok)
	s.Require().Equal(http.StatusNotFound, response.Code)
	s.Require().Equal("заказ не найден", response.Message)
}

func (s *APISuite) TestCancelOrderInvalidStatus() {
	var (
		orderUUID = uuid.New()

		params = orderv1.CancelOrderParams{
			OrderUUID: orderUUID,
		}
	)

	s.orderService.
		EXPECT().
		Cancel(s.ctx, orderUUID).
		Return(errs.ErrInvalidOrderStatus)

	res, err := s.api.CancelOrder(s.ctx, params)

	s.Require().NoError(err)
	s.Require().NotNil(res)

	response, ok := res.(*orderv1.CancelOrderConflict)

	s.Require().True(ok)
	s.Require().Equal(http.StatusConflict, response.Code)
	s.Require().Equal("неверный статус заказа", response.Message)
}

func (s *APISuite) TestCancelOrderInternalError() {
	var (
		orderUUID  = uuid.New()
		serviceErr = errors.New(gofakeit.Sentence(5))

		params = orderv1.CancelOrderParams{
			OrderUUID: orderUUID,
		}
	)

	s.orderService.
		EXPECT().
		Cancel(s.ctx, orderUUID).
		Return(serviceErr)

	res, err := s.api.CancelOrder(s.ctx, params)

	s.Require().NoError(err)
	s.Require().NotNil(res)

	response, ok := res.(*orderv1.CancelOrderInternalServerError)

	s.Require().True(ok)
	s.Require().Equal(http.StatusInternalServerError, response.Code)
	s.Require().Equal("непоправимая ошибка", response.Message)
}
