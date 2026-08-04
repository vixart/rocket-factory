package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vixart/rocket-factory/payment/internal/api/payment/v1"
	"github.com/vixart/rocket-factory/payment/internal/api/payment/v1/mocks"
	errs "github.com/vixart/rocket-factory/payment/internal/errors"
	"github.com/vixart/rocket-factory/payment/internal/model"
	paymentv1 "github.com/vixart/rocket-factory/shared/pkg/proto/payment/v1"
)

func TestPayOrder(t *testing.T) {
	t.Parallel()

	type args struct {
		req *paymentv1.PayOrderRequest
	}

	var (
		ctx = context.Background()

		validUUID = uuid.New()
		txUUID    = uuid.New()

		internalErr = errors.New("internal error")
	)

	tests := []struct {
		name      string
		args      args
		setupMock func(service *mocks.PaymentService)
		check     func(t *testing.T, res *paymentv1.PayOrderResponse, err error)
	}{
		{
			name: "empty order_uuid",
			args: args{
				req: &paymentv1.PayOrderRequest{
					OrderUuid: "",
				},
			},
			setupMock: func(service *mocks.PaymentService) {},
			check: func(t *testing.T, res *paymentv1.PayOrderResponse, err error) {
				require.Error(t, err)
				require.Nil(t, res)

				assert.ErrorIs(t, err, errs.ErrInvalidUUID)
			},
		},
		{
			name: "invalid order_uuid",
			args: args{
				req: &paymentv1.PayOrderRequest{
					OrderUuid: "not-a-uuid",
				},
			},
			setupMock: func(service *mocks.PaymentService) {},
			check: func(t *testing.T, res *paymentv1.PayOrderResponse, err error) {
				require.Error(t, err)
				require.Nil(t, res)

				assert.ErrorIs(t, err, errs.ErrInvalidUUID)
			},
		},
		{
			name: "payment service fails",
			args: args{
				req: &paymentv1.PayOrderRequest{
					OrderUuid:     validUUID.String(),
					PaymentMethod: paymentv1.PaymentMethod_PAYMENT_METHOD_CARD,
				},
			},
			setupMock: func(service *mocks.PaymentService) {
				service.EXPECT().
					PayOrder(
						ctx,
						validUUID,
						model.PaymentMethodCard,
					).
					Return((*uuid.UUID)(nil), internalErr)
			},
			check: func(t *testing.T, res *paymentv1.PayOrderResponse, err error) {
				require.Error(t, err)
				require.Nil(t, res)

				assert.ErrorIs(t, err, internalErr)
			},
		},
		{
			name: "order paid successfully",
			args: args{
				req: &paymentv1.PayOrderRequest{
					OrderUuid:     validUUID.String(),
					PaymentMethod: paymentv1.PaymentMethod_PAYMENT_METHOD_CARD,
				},
			},
			setupMock: func(service *mocks.PaymentService) {
				service.EXPECT().
					PayOrder(
						ctx,
						validUUID,
						model.PaymentMethodCard,
					).
					Return(&txUUID, nil)
			},
			check: func(t *testing.T, res *paymentv1.PayOrderResponse, err error) {
				require.NoError(t, err)
				require.NotNil(t, res)

				assert.Equal(t, txUUID.String(), res.TransactionUuid)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			paymentService := mocks.NewPaymentService(t)

			tc.setupMock(paymentService)

			api := v1.NewApi(paymentService)

			res, err := api.PayOrder(ctx, tc.args.req)

			tc.check(t, res, err)
		})
	}
}
