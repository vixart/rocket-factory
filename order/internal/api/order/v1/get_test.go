package v1

import (
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"

	errs "github.com/vixart/rocket-factory/order/internal/errors"
	"github.com/vixart/rocket-factory/order/internal/model"
	orderv1 "github.com/vixart/rocket-factory/shared/pkg/openapi/order/v1"
)

func (s *APISuite) TestGetOrderSuccess() {
	var (
		orderUUID       = uuid.New()
		hullUUID        = uuid.New()
		engineUUID      = uuid.New()
		shieldUUID      = uuid.New()
		weaponUUID      = uuid.New()
		transactionUUID = uuid.New()

		paymentMethod = model.PaymentMethodCard

		createdAt = time.Now()

		order = model.Order{
			OrderUUID:       orderUUID,
			HullUUID:        hullUUID,
			EngineUUID:      engineUUID,
			ShieldUUID:      &shieldUUID,
			WeaponUUID:      &weaponUUID,
			TotalPrice:      gofakeit.Int64(),
			TransactionUUID: &transactionUUID,
			PaymentMethod:   &paymentMethod,
			Status:          model.OrderStatusPaid,
			CreatedAt:       createdAt,
		}

		params = orderv1.GetOrderParams{
			OrderUUID: orderUUID,
		}
	)

	s.orderService.
		EXPECT().
		Get(s.ctx, orderUUID).
		Return(order, nil)

	res, err := s.api.GetOrder(s.ctx, params)

	s.Require().NoError(err)
	s.Require().NotNil(res)

	orderRes, ok := res.(*orderv1.OrderDto)

	s.Require().True(ok)

	s.Require().Equal(orderUUID, orderRes.OrderUUID)
	s.Require().Equal(hullUUID, orderRes.HullUUID)
	s.Require().Equal(engineUUID, orderRes.EngineUUID)

	s.Require().True(orderRes.ShieldUUID.Set)
	s.Require().Equal(shieldUUID, orderRes.ShieldUUID.Value)

	s.Require().True(orderRes.WeaponUUID.Set)
	s.Require().Equal(weaponUUID, orderRes.WeaponUUID.Value)

	s.Require().Equal(order.TotalPrice, orderRes.TotalPrice)

	s.Require().True(orderRes.TransactionUUID.Set)
	s.Require().Equal(transactionUUID, orderRes.TransactionUUID.Value)

	s.Require().True(orderRes.PaymentMethod.Set)
	s.Require().Equal(
		orderv1.PaymentMethod(paymentMethod),
		orderRes.PaymentMethod.Value,
	)

	s.Require().Equal(
		orderv1.OrderStatus(order.Status),
		orderRes.Status,
	)

	s.Require().Equal(createdAt, orderRes.CreatedAt)
}

func (s *APISuite) TestGetOrderSuccessWithoutOptionalFields() {
	var (
		orderUUID  = uuid.New()
		hullUUID   = uuid.New()
		engineUUID = uuid.New()

		createdAt = time.Now()

		order = model.Order{
			OrderUUID:  orderUUID,
			HullUUID:   hullUUID,
			EngineUUID: engineUUID,
			TotalPrice: gofakeit.Int64(),
			Status:     model.OrderStatusPendingPayment,
			CreatedAt:  createdAt,
		}

		params = orderv1.GetOrderParams{
			OrderUUID: orderUUID,
		}
	)

	s.orderService.
		EXPECT().
		Get(s.ctx, orderUUID).
		Return(order, nil)

	res, err := s.api.GetOrder(s.ctx, params)

	s.Require().NoError(err)
	s.Require().NotNil(res)

	orderRes, ok := res.(*orderv1.OrderDto)

	s.Require().True(ok)

	s.Require().False(orderRes.ShieldUUID.Set)
	s.Require().False(orderRes.WeaponUUID.Set)
	s.Require().False(orderRes.TransactionUUID.Set)
	s.Require().False(orderRes.PaymentMethod.Set)
}

func (s *APISuite) TestGetOrderNotFound() {
	var (
		orderUUID = uuid.New()

		params = orderv1.GetOrderParams{
			OrderUUID: orderUUID,
		}
	)

	s.orderService.
		EXPECT().
		Get(s.ctx, orderUUID).
		Return(model.Order{}, errs.ErrOrderNotFound)

	res, err := s.api.GetOrder(s.ctx, params)

	s.Require().NoError(err)
	s.Require().NotNil(res)

	notFoundRes, ok := res.(*orderv1.GetOrderNotFound)

	s.Require().True(ok)
	s.Require().Equal(404, notFoundRes.Code)
	s.Require().Equal("заказ не найден", notFoundRes.Message)
}

func (s *APISuite) TestGetOrderInternalError() {
	var (
		orderUUID  = uuid.New()
		serviceErr = gofakeit.Error()

		params = orderv1.GetOrderParams{
			OrderUUID: orderUUID,
		}
	)

	s.orderService.
		EXPECT().
		Get(s.ctx, orderUUID).
		Return(model.Order{}, serviceErr)

	res, err := s.api.GetOrder(s.ctx, params)

	s.Require().NoError(err)
	s.Require().NotNil(res)

	internalRes, ok := res.(*orderv1.GetOrderInternalServerError)

	s.Require().True(ok)
	s.Require().Equal(500, internalRes.Code)
	s.Require().Equal("что-то пошло не так", internalRes.Message)
}
