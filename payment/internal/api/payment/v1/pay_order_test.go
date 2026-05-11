package v1

import (
	"errors"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/vixart/rocket-factory/payment/internal/api/payment/converter"
	paymentv1 "github.com/vixart/rocket-factory/shared/pkg/proto/payment/v1"
)

func (s *APISuite) TestPayOrderSuccess() {
	var (
		orderUUID = uuid.New()
		txUUID    = uuid.New()

		req = &paymentv1.PayOrderRequest{
			OrderUuid:     orderUUID.String(),
			PaymentMethod: paymentv1.PaymentMethod_PAYMENT_METHOD_CARD,
		}

		expectedPaymentMethod = converter.PaymentMethodProtoToModel(
			req.GetPaymentMethod(),
		)
	)

	s.paymentService.
		EXPECT().
		PayOrder(s.ctx, orderUUID, expectedPaymentMethod).
		Return(&txUUID, nil)

	res, err := s.api.PayOrder(s.ctx, req)

	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Require().Equal(txUUID.String(), res.GetTransactionUuid())
}

func (s *APISuite) TestPayOrderEmptyOrderUUID() {
	req := &paymentv1.PayOrderRequest{
		OrderUuid: "",
	}

	res, err := s.api.PayOrder(s.ctx, req)

	s.Require().Error(err)
	s.Require().Nil(res)

	st, ok := status.FromError(err)

	s.Require().True(ok)
	s.Require().Equal(codes.InvalidArgument, st.Code())
	s.Require().Equal(
		"order_uuid не может быть пустым",
		st.Message(),
	)
}

func (s *APISuite) TestPayOrderInvalidUUID() {
	req := &paymentv1.PayOrderRequest{
		OrderUuid: "invalid-uuid",
	}

	res, err := s.api.PayOrder(s.ctx, req)

	s.Require().Error(err)
	s.Require().Nil(res)

	st, ok := status.FromError(err)

	s.Require().True(ok)
	s.Require().Equal(codes.InvalidArgument, st.Code())
}

func (s *APISuite) TestPayOrderServiceError() {
	var (
		orderUUID = uuid.New()

		serviceErr = errors.New(gofakeit.Sentence(5))

		req = &paymentv1.PayOrderRequest{
			OrderUuid:     orderUUID.String(),
			PaymentMethod: paymentv1.PaymentMethod_PAYMENT_METHOD_CARD,
		}

		expectedPaymentMethod = converter.PaymentMethodProtoToModel(
			req.GetPaymentMethod(),
		)
	)

	s.paymentService.
		EXPECT().
		PayOrder(s.ctx, orderUUID, expectedPaymentMethod).
		Return(nil, serviceErr)

	res, err := s.api.PayOrder(s.ctx, req)

	s.Require().Error(err)
	s.Require().Nil(res)

	s.Require().ErrorIs(err, serviceErr)
	s.Require().Equal(serviceErr.Error(), err.Error())
}
