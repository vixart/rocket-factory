package order

import (
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/order/internal/model"
)

func (s *ServiceSuite) TestGetSuccess() {
	var (
		orderUUID  = uuid.New()
		hullUUID   = uuid.New()
		engineUUID = uuid.New()
		createdAt  = time.Now()

		order = &model.Order{
			OrderUUID:       orderUUID,
			HullUUID:        hullUUID,
			EngineUUID:      engineUUID,
			ShieldUUID:      new(uuid.New()),
			WeaponUUID:      new(uuid.New()),
			TotalPrice:      250000,
			TransactionUUID: new(uuid.New()),
			PaymentMethod:   new(model.PaymentMethodCard),
			Status:          model.OrderStatusPaid,
			CreatedAt:       createdAt,
		}
	)

	s.orderRepo.
		EXPECT().
		Get(s.ctx, orderUUID).
		Return(order, nil)

	res, err := s.service.Get(s.ctx, orderUUID)

	s.Require().NoError(err)
	s.Require().Equal(*order, res)
}

func (s *ServiceSuite) TestGetRepositoryError() {
	var (
		orderUUID = uuid.New()
		repoErr   = gofakeit.Error()
	)

	s.orderRepo.
		EXPECT().
		Get(s.ctx, orderUUID).
		Return(nil, repoErr)

	res, err := s.service.Get(s.ctx, orderUUID)

	s.Require().ErrorIs(err, repoErr)
	s.Require().Equal(model.Order{}, res)
}
