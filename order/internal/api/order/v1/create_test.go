package v1

import (
	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"

	errs "github.com/vixart/rocket-factory/order/internal/errors"
	"github.com/vixart/rocket-factory/order/internal/model"
	orderv1 "github.com/vixart/rocket-factory/shared/pkg/openapi/order/v1"
)

func (s *APISuite) TestCreateOrderSuccess() {
	var (
		engineUUID = uuid.New()
		hullUUID   = uuid.New()
		shieldUUID = uuid.New()
		weaponUUID = uuid.New()

		orderUUID = uuid.New()

		req = &orderv1.CreateOrderRequest{
			EngineUUID: engineUUID,
			HullUUID:   hullUUID,
			ShieldUUID: orderv1.NewOptNilUUID(shieldUUID),
			WeaponUUID: orderv1.NewOptNilUUID(weaponUUID),
		}

		expectedOrderParts = model.OrderParts{
			EngineUUID: engineUUID,
			HullUUID:   hullUUID,
			ShieldUUID: &shieldUUID,
			WeaponUUID: &weaponUUID,
		}

		expectedOrder = &model.Order{
			OrderUUID:  orderUUID,
			HullUUID:   hullUUID,
			EngineUUID: engineUUID,
			ShieldUUID: &shieldUUID,
			WeaponUUID: &weaponUUID,
			TotalPrice: gofakeit.Int64(),
			Status:     model.OrderStatusPendingPayment,
		}
	)

	s.orderService.
		EXPECT().
		Create(s.ctx, expectedOrderParts).
		Return(expectedOrder, nil)

	res, err := s.api.CreateOrder(s.ctx, req)

	s.Require().NoError(err)
	s.Require().NotNil(res)

	successRes, ok := res.(*orderv1.CreateOrderResponse)

	s.Require().True(ok)
	s.Require().Equal(orderUUID, successRes.OrderUUID)
	s.Require().Equal(expectedOrder.TotalPrice, successRes.TotalPrice)
}

func (s *APISuite) TestCreateOrderSuccessWithoutOptionalParts() {
	var (
		engineUUID = uuid.New()
		hullUUID   = uuid.New()

		orderUUID = uuid.New()

		req = &orderv1.CreateOrderRequest{
			EngineUUID: engineUUID,
			HullUUID:   hullUUID,
		}

		expectedOrderParts = model.OrderParts{
			EngineUUID: engineUUID,
			HullUUID:   hullUUID,
		}

		expectedOrder = &model.Order{
			OrderUUID:  orderUUID,
			HullUUID:   hullUUID,
			EngineUUID: engineUUID,
			TotalPrice: gofakeit.Int64(),
			Status:     model.OrderStatusPendingPayment,
		}
	)

	s.orderService.
		EXPECT().
		Create(s.ctx, expectedOrderParts).
		Return(expectedOrder, nil)

	res, err := s.api.CreateOrder(s.ctx, req)

	s.Require().NoError(err)
	s.Require().NotNil(res)

	successRes, ok := res.(*orderv1.CreateOrderResponse)

	s.Require().True(ok)
	s.Require().Equal(orderUUID, successRes.OrderUUID)
	s.Require().Equal(expectedOrder.TotalPrice, successRes.TotalPrice)
}

func (s *APISuite) TestCreateOrderPartNotFound() {
	var (
		engineUUID = uuid.New()
		hullUUID   = uuid.New()

		req = &orderv1.CreateOrderRequest{
			EngineUUID: engineUUID,
			HullUUID:   hullUUID,
		}

		expectedOrderParts = model.OrderParts{
			EngineUUID: engineUUID,
			HullUUID:   hullUUID,
		}

		serviceErr = errs.ErrPartNotFound
	)

	s.orderService.
		EXPECT().
		Create(s.ctx, expectedOrderParts).
		Return(nil, serviceErr)

	res, err := s.api.CreateOrder(s.ctx, req)

	s.Require().NoError(err)
	s.Require().NotNil(res)

	notFoundRes, ok := res.(*orderv1.CreateOrderNotFound)

	s.Require().True(ok)
	s.Require().Equal(404, notFoundRes.Code)
	s.Require().Equal(serviceErr.Error(), notFoundRes.Message)
}

func (s *APISuite) TestCreateOrderInventoryPartNotFound() {
	var (
		engineUUID = uuid.New()
		hullUUID   = uuid.New()

		req = &orderv1.CreateOrderRequest{
			EngineUUID: engineUUID,
			HullUUID:   hullUUID,
		}

		expectedOrderParts = model.OrderParts{
			EngineUUID: engineUUID,
			HullUUID:   hullUUID,
		}

		serviceErr = errs.ErrInventoryPartNotFound
	)

	s.orderService.
		EXPECT().
		Create(s.ctx, expectedOrderParts).
		Return(nil, serviceErr)

	res, err := s.api.CreateOrder(s.ctx, req)

	s.Require().NoError(err)
	s.Require().NotNil(res)

	notFoundRes, ok := res.(*orderv1.CreateOrderNotFound)

	s.Require().True(ok)
	s.Require().Equal(404, notFoundRes.Code)
	s.Require().Equal(serviceErr.Error(), notFoundRes.Message)
}

func (s *APISuite) TestCreateOrderInvalidUUID() {
	var (
		engineUUID = uuid.New()
		hullUUID   = uuid.New()

		req = &orderv1.CreateOrderRequest{
			EngineUUID: engineUUID,
			HullUUID:   hullUUID,
		}

		expectedOrderParts = model.OrderParts{
			EngineUUID: engineUUID,
			HullUUID:   hullUUID,
		}

		serviceErr = errs.ErrInvalidUUID
	)

	s.orderService.
		EXPECT().
		Create(s.ctx, expectedOrderParts).
		Return(nil, serviceErr)

	res, err := s.api.CreateOrder(s.ctx, req)

	s.Require().NoError(err)
	s.Require().NotNil(res)

	badRequestRes, ok := res.(*orderv1.CreateOrderBadRequest)

	s.Require().True(ok)
	s.Require().Equal(400, badRequestRes.Code)
	s.Require().Equal(serviceErr.Error(), badRequestRes.Message)
}

func (s *APISuite) TestCreateOrderAlreadyExists() {
	var (
		engineUUID = uuid.New()
		hullUUID   = uuid.New()

		req = &orderv1.CreateOrderRequest{
			EngineUUID: engineUUID,
			HullUUID:   hullUUID,
		}

		expectedOrderParts = model.OrderParts{
			EngineUUID: engineUUID,
			HullUUID:   hullUUID,
		}

		serviceErr = errs.ErrOrderAlreadyExists
	)

	s.orderService.
		EXPECT().
		Create(s.ctx, expectedOrderParts).
		Return(nil, serviceErr)

	res, err := s.api.CreateOrder(s.ctx, req)

	s.Require().NoError(err)
	s.Require().NotNil(res)

	badRequestRes, ok := res.(*orderv1.CreateOrderBadRequest)

	s.Require().True(ok)
	s.Require().Equal(400, badRequestRes.Code)
	s.Require().Equal(serviceErr.Error(), badRequestRes.Message)
}

func (s *APISuite) TestCreateOrderInsufficientStock() {
	var (
		engineUUID = uuid.New()
		hullUUID   = uuid.New()

		req = &orderv1.CreateOrderRequest{
			EngineUUID: engineUUID,
			HullUUID:   hullUUID,
		}

		expectedOrderParts = model.OrderParts{
			EngineUUID: engineUUID,
			HullUUID:   hullUUID,
		}

		serviceErr = errs.ErrPartInsufficientStock
	)

	s.orderService.
		EXPECT().
		Create(s.ctx, expectedOrderParts).
		Return(nil, serviceErr)

	res, err := s.api.CreateOrder(s.ctx, req)

	s.Require().NoError(err)
	s.Require().NotNil(res)

	conflictRes, ok := res.(*orderv1.CreateOrderConflict)

	s.Require().True(ok)
	s.Require().Equal(409, conflictRes.Code)
	s.Require().Equal(serviceErr.Error(), conflictRes.Message)
}

func (s *APISuite) TestCreateOrderInternalError() {
	var (
		engineUUID = uuid.New()
		hullUUID   = uuid.New()

		req = &orderv1.CreateOrderRequest{
			EngineUUID: engineUUID,
			HullUUID:   hullUUID,
		}

		expectedOrderParts = model.OrderParts{
			EngineUUID: engineUUID,
			HullUUID:   hullUUID,
		}

		serviceErr = gofakeit.Error()
	)

	s.orderService.
		EXPECT().
		Create(s.ctx, expectedOrderParts).
		Return(nil, serviceErr)

	res, err := s.api.CreateOrder(s.ctx, req)

	s.Require().NoError(err)
	s.Require().NotNil(res)

	internalRes, ok := res.(*orderv1.CreateOrderInternalServerError)

	s.Require().True(ok)
	s.Require().Equal(500, internalRes.Code)
	s.Require().Equal("непоправимая ошибка", internalRes.Message)
}
