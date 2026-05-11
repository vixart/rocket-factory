package order

import (
	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	errs "github.com/vixart/rocket-factory/order/internal/errors"
	"github.com/vixart/rocket-factory/order/internal/model"
)

func (s *ServiceSuite) TestCancelSuccess() {
	var (
		orderUUID = uuid.New()

		order = &model.Order{
			OrderUUID:  orderUUID,
			HullUUID:   uuid.New(),
			EngineUUID: uuid.New(),
			TotalPrice: gofakeit.Int64(),
			Status:     model.OrderStatusPendingPayment,
		}
	)

	s.orderRepo.
		EXPECT().
		Get(s.ctx, orderUUID).
		Return(order, nil)

	s.orderRepo.
		EXPECT().
		Update(s.ctx, mock.MatchedBy(func(updated model.Order) bool {
			return updated.OrderUUID == orderUUID &&
				updated.Status == model.OrderStatusCanceled
		})).
		Return(nil)

	err := s.service.Cancel(s.ctx, orderUUID)

	s.NoError(err)
}

func (s *ServiceSuite) TestCancelOrderNotFound() {
	var (
		orderUUID = uuid.New()
		repoErr   = gofakeit.Error()
	)

	s.orderRepo.
		EXPECT().
		Get(s.ctx, orderUUID).
		Return(nil, repoErr)

	err := s.service.Cancel(s.ctx, orderUUID)

	s.Error(err)
	s.ErrorIs(err, repoErr)
}

func (s *ServiceSuite) TestCancelInvalidStatusPaid() {
	var (
		orderUUID = uuid.New()

		order = &model.Order{
			OrderUUID:  orderUUID,
			HullUUID:   uuid.New(),
			EngineUUID: uuid.New(),
			TotalPrice: gofakeit.Int64(),
			Status:     model.OrderStatusPaid,
		}
	)

	s.orderRepo.
		EXPECT().
		Get(s.ctx, orderUUID).
		Return(order, nil)

	err := s.service.Cancel(s.ctx, orderUUID)

	s.Error(err)
	s.ErrorIs(err, errs.ErrInvalidOrderStatus)
}

func (s *ServiceSuite) TestCancelInvalidStatusCanceled() {
	var (
		orderUUID = uuid.New()

		order = &model.Order{
			OrderUUID:  orderUUID,
			HullUUID:   uuid.New(),
			EngineUUID: uuid.New(),
			TotalPrice: gofakeit.Int64(),
			Status:     model.OrderStatusCanceled,
		}
	)

	s.orderRepo.
		EXPECT().
		Get(s.ctx, orderUUID).
		Return(order, nil)

	err := s.service.Cancel(s.ctx, orderUUID)

	s.Error(err)
	s.ErrorIs(err, errs.ErrInvalidOrderStatus)
}

func (s *ServiceSuite) TestCancelUpdateError() {
	var (
		orderUUID = uuid.New()
		updateErr = gofakeit.Error()

		order = &model.Order{
			OrderUUID:  orderUUID,
			HullUUID:   uuid.New(),
			EngineUUID: uuid.New(),
			TotalPrice: gofakeit.Int64(),
			Status:     model.OrderStatusPendingPayment,
		}
	)

	s.orderRepo.
		EXPECT().
		Get(s.ctx, orderUUID).
		Return(order, nil)

	s.orderRepo.
		EXPECT().
		Update(s.ctx, mock.MatchedBy(func(updated model.Order) bool {
			return updated.OrderUUID == orderUUID &&
				updated.Status == model.OrderStatusCanceled
		})).
		Return(updateErr)

	err := s.service.Cancel(s.ctx, orderUUID)

	s.Error(err)
	s.ErrorIs(err, updateErr)
}
