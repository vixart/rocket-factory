package order

import (
	"errors"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	errs "github.com/vixart/rocket-factory/order/internal/errors"
	"github.com/vixart/rocket-factory/order/internal/model"
)

func (s *ServiceSuite) TestCreateSuccessRequiredPartsOnly() {
	var (
		hullUUID   = uuid.New()
		engineUUID = uuid.New()

		orderParts = model.OrderParts{
			HullUUID:   hullUUID,
			EngineUUID: engineUUID,
		}

		hullPart = &model.Part{
			UUID:          hullUUID,
			Price:         100000,
			StockQuantity: 5,
		}

		enginePart = &model.Part{
			UUID:          engineUUID,
			Price:         50000,
			StockQuantity: 3,
		}

		parts = map[uuid.UUID]*model.Part{
			hullUUID:   hullPart,
			engineUUID: enginePart,
		}
	)

	s.inventoryClient.
		EXPECT().
		ListParts(mock.Anything, []uuid.UUID{hullUUID, engineUUID}).
		Return(parts, nil)

	s.orderRepo.
		EXPECT().
		Create(mock.Anything, mock.AnythingOfType("model.Order")).
		Return(nil)

	res, err := s.service.Create(s.ctx, orderParts)

	s.Require().NoError(err)
	s.Require().NotNil(res)

	s.Require().Equal(hullUUID, res.HullUUID)
	s.Require().Equal(engineUUID, res.EngineUUID)
	s.Require().Equal(int64(150000), res.TotalPrice)
	s.Require().Equal(model.OrderStatusPendingPayment, res.Status)
}

func (s *ServiceSuite) TestCreateSuccessAllParts() {
	var (
		hullUUID   = uuid.New()
		engineUUID = uuid.New()
		shieldUUID = uuid.New()
		weaponUUID = uuid.New()

		orderParts = model.OrderParts{
			HullUUID:   hullUUID,
			EngineUUID: engineUUID,
			ShieldUUID: &shieldUUID,
			WeaponUUID: &weaponUUID,
		}

		hullPart = &model.Part{
			UUID:          hullUUID,
			Price:         100000,
			StockQuantity: 5,
		}

		enginePart = &model.Part{
			UUID:          engineUUID,
			Price:         50000,
			StockQuantity: 3,
		}

		shieldPart = &model.Part{
			UUID:          shieldUUID,
			Price:         25000,
			StockQuantity: 2,
		}

		weaponPart = &model.Part{
			UUID:          weaponUUID,
			Price:         30000,
			StockQuantity: 1,
		}

		parts = map[uuid.UUID]*model.Part{
			hullUUID:   hullPart,
			engineUUID: enginePart,
			shieldUUID: shieldPart,
			weaponUUID: weaponPart,
		}
	)

	s.inventoryClient.
		EXPECT().
		ListParts(
			mock.Anything,
			[]uuid.UUID{hullUUID, engineUUID, shieldUUID, weaponUUID},
		).
		Return(parts, nil)

	s.orderRepo.
		EXPECT().
		Create(mock.Anything, mock.AnythingOfType("model.Order")).
		Return(nil)

	res, err := s.service.Create(s.ctx, orderParts)

	s.Require().NoError(err)
	s.Require().NotNil(res)

	s.Require().Equal(int64(205000), res.TotalPrice)

	s.Require().NotNil(res.ShieldUUID)
	s.Require().NotNil(res.WeaponUUID)

	s.Require().Equal(shieldUUID, *res.ShieldUUID)

	s.Require().Equal(weaponUUID, *res.WeaponUUID)
}

func (s *ServiceSuite) TestCreateInventoryClientError() {
	var (
		hullUUID   = uuid.New()
		engineUUID = uuid.New()

		clientErr = errors.New("inventory unavailable")

		orderParts = model.OrderParts{
			HullUUID:   hullUUID,
			EngineUUID: engineUUID,
		}
	)

	s.inventoryClient.
		EXPECT().
		ListParts(mock.Anything, []uuid.UUID{hullUUID, engineUUID}).
		Return(nil, clientErr)

	res, err := s.service.Create(s.ctx, orderParts)

	s.Require().Error(err)
	s.Require().ErrorContains(err, clientErr.Error())
	s.Require().Nil(res)
}

func (s *ServiceSuite) TestCreatePartOutOfStock() {
	var (
		hullUUID   = uuid.New()
		engineUUID = uuid.New()

		orderParts = model.OrderParts{
			HullUUID:   hullUUID,
			EngineUUID: engineUUID,
		}

		hullPart = &model.Part{
			UUID:          hullUUID,
			Price:         100000,
			StockQuantity: 0,
		}

		enginePart = &model.Part{
			UUID:          engineUUID,
			Price:         50000,
			StockQuantity: 5,
		}

		parts = map[uuid.UUID]*model.Part{
			hullUUID:   hullPart,
			engineUUID: enginePart,
		}
	)

	s.inventoryClient.
		EXPECT().
		ListParts(mock.Anything, []uuid.UUID{hullUUID, engineUUID}).
		Return(parts, nil)

	res, err := s.service.Create(s.ctx, orderParts)

	s.Require().Error(err)
	s.Require().ErrorIs(err, errs.ErrPartInsufficientStock)
	s.Require().Nil(res)
}

func (s *ServiceSuite) TestCreateEnginePartNotFound() {
	var (
		hullUUID   = uuid.New()
		engineUUID = uuid.New()

		orderParts = model.OrderParts{
			HullUUID:   hullUUID,
			EngineUUID: engineUUID,
		}

		hullPart = &model.Part{
			UUID:          hullUUID,
			Price:         100000,
			StockQuantity: 5,
		}

		parts = map[uuid.UUID]*model.Part{
			hullUUID: hullPart,
		}
	)

	s.inventoryClient.
		EXPECT().
		ListParts(mock.Anything, []uuid.UUID{hullUUID, engineUUID}).
		Return(parts, nil)

	res, err := s.service.Create(s.ctx, orderParts)

	s.Require().Error(err)
	s.Require().ErrorIs(err, errs.ErrPartNotFound)
	s.Require().Nil(res)
}

func (s *ServiceSuite) TestCreateRepositoryError() {
	var (
		hullUUID   = uuid.New()
		engineUUID = uuid.New()

		repoErr = gofakeit.Error()

		orderParts = model.OrderParts{
			HullUUID:   hullUUID,
			EngineUUID: engineUUID,
		}

		hullPart = &model.Part{
			UUID:          hullUUID,
			Price:         100000,
			StockQuantity: 5,
		}

		enginePart = &model.Part{
			UUID:          engineUUID,
			Price:         50000,
			StockQuantity: 3,
		}

		parts = map[uuid.UUID]*model.Part{
			hullUUID:   hullPart,
			engineUUID: enginePart,
		}
	)

	s.inventoryClient.
		EXPECT().
		ListParts(mock.Anything, []uuid.UUID{hullUUID, engineUUID}).
		Return(parts, nil)

	s.orderRepo.
		EXPECT().
		Create(mock.Anything, mock.AnythingOfType("model.Order")).
		Return(repoErr)

	res, err := s.service.Create(s.ctx, orderParts)

	s.Require().ErrorIs(err, repoErr)
	s.Require().Nil(res)
}
