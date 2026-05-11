package order

import (
	"errors"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	errs "github.com/vixart/rocket-factory/order/internal/errors"
	"github.com/vixart/rocket-factory/order/internal/model"
)

func (s *ServiceSuite) TestPaySuccess() {
	var (
		orderUUID     = uuid.New()
		hullUUID      = uuid.New()
		engineUUID    = uuid.New()
		transactionID = uuid.New()
		paymentMethod = model.PaymentMethodCard
		createdAt     = time.Now()

		order = &model.Order{
			OrderUUID:  orderUUID,
			HullUUID:   hullUUID,
			EngineUUID: engineUUID,
			TotalPrice: 150000,
			Status:     model.OrderStatusPendingPayment,
			CreatedAt:  createdAt,
		}
	)

	s.orderRepo.
		EXPECT().
		Get(mock.Anything, orderUUID).
		Return(order, nil)

	s.paymentClient.
		EXPECT().
		PayOrder(mock.Anything, orderUUID, paymentMethod).
		Return(&transactionID, nil)

	s.orderRepo.
		EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(o model.Order) bool {
			return o.OrderUUID == orderUUID &&
				o.Status == model.OrderStatusPaid &&
				o.TransactionUUID != nil &&
				o.PaymentMethod != nil
		})).
		Return(nil)

	res, err := s.service.Pay(s.ctx, orderUUID, paymentMethod)

	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Require().Equal(transactionID, *res)

	s.Require().Equal(model.OrderStatusPaid, order.Status)
	s.Require().NotNil(order.PaymentMethod)
	s.Require().Equal(paymentMethod, *order.PaymentMethod)

	s.Require().NotNil(order.TransactionUUID)
	s.Require().Equal(transactionID, *order.TransactionUUID)
}

func (s *ServiceSuite) TestPayInvalidPaymentMethod() {
	orderUUID := uuid.New()

	res, err := s.service.Pay(
		s.ctx,
		orderUUID,
		model.PaymentMethodUnspecified,
	)

	s.Require().ErrorIs(err, errs.ErrInvalidPaymentMethod)
	s.Require().Nil(res)
}

func (s *ServiceSuite) TestPayOrderRepositoryGetError() {
	var (
		orderUUID = uuid.New()
		repoErr   = gofakeit.Error()
	)

	s.orderRepo.
		EXPECT().
		Get(mock.Anything, orderUUID).
		Return(nil, repoErr)

	res, err := s.service.Pay(
		s.ctx,
		orderUUID,
		model.PaymentMethodCard,
	)

	s.Require().ErrorIs(err, repoErr)
	s.Require().Nil(res)
}

func (s *ServiceSuite) TestPayCanceledOrder() {
	var (
		orderUUID  = uuid.New()
		hullUUID   = uuid.New()
		engineUUID = uuid.New()
		createdAt  = time.Now()

		order = &model.Order{
			OrderUUID:  orderUUID,
			HullUUID:   hullUUID,
			EngineUUID: engineUUID,
			TotalPrice: 100000,
			Status:     model.OrderStatusCanceled,
			CreatedAt:  createdAt,
		}
	)

	s.orderRepo.
		EXPECT().
		Get(mock.Anything, orderUUID).
		Return(order, nil)

	res, err := s.service.Pay(
		s.ctx,
		orderUUID,
		model.PaymentMethodCard,
	)

	s.Require().ErrorIs(err, errs.ErrInvalidOrderStatus)
	s.Require().Nil(res)
}

func (s *ServiceSuite) TestPayPaymentClientError() {
	var (
		orderUUID     = uuid.New()
		hullUUID      = uuid.New()
		engineUUID    = uuid.New()
		paymentMethod = model.PaymentMethodCard
		paymentErr    = errors.New("payment failed")
		createdAt     = time.Now()

		order = &model.Order{
			OrderUUID:  orderUUID,
			HullUUID:   hullUUID,
			EngineUUID: engineUUID,
			TotalPrice: 120000,
			Status:     model.OrderStatusPendingPayment,
			CreatedAt:  createdAt,
		}
	)

	s.orderRepo.
		EXPECT().
		Get(mock.Anything, orderUUID).
		Return(order, nil)

	s.paymentClient.
		EXPECT().
		PayOrder(mock.Anything, orderUUID, paymentMethod).
		Return(nil, paymentErr)

	res, err := s.service.Pay(
		s.ctx,
		orderUUID,
		paymentMethod,
	)

	s.Require().ErrorIs(err, paymentErr)
	s.Require().Nil(res)
}

func (s *ServiceSuite) TestPayOrderRepositoryUpdateError() {
	var (
		orderUUID  = uuid.New()
		hullUUID   = uuid.New()
		engineUUID = uuid.New()

		paymentMethod = model.PaymentMethodCard
		updateErr     = errors.New("update failed")
		createdAt     = time.Now()

		order = &model.Order{
			OrderUUID:  orderUUID,
			HullUUID:   hullUUID,
			EngineUUID: engineUUID,
			TotalPrice: 90000,
			Status:     model.OrderStatusPendingPayment,
			CreatedAt:  createdAt,
		}
	)

	s.orderRepo.
		EXPECT().
		Get(mock.Anything, orderUUID).
		Return(order, nil)

	s.paymentClient.
		EXPECT().
		PayOrder(mock.Anything, orderUUID, paymentMethod).
		Return(new(uuid.New()), nil)

	s.orderRepo.
		EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(o model.Order) bool {
			return o.Status == model.OrderStatusPaid &&
				o.TransactionUUID != nil &&
				o.PaymentMethod != nil
		})).
		Return(updateErr)

	res, err := s.service.Pay(
		s.ctx,
		orderUUID,
		paymentMethod,
	)

	s.Require().ErrorIs(err, updateErr)
	s.Require().Nil(res)
}
