package v1

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/vixart/rocket-factory/payment/internal/api/payment/v1/mocks"
	errs "github.com/vixart/rocket-factory/payment/internal/errors"
	paymentv1 "github.com/vixart/rocket-factory/shared/pkg/proto/payment/v1"
)

func TestPayOrder(t *testing.T) {
	t.Parallel()

	type args struct {
		req *paymentv1.PayOrderRequest
	}

	type expected struct {
		errCode codes.Code
		err     error
	}

	ctx := context.Background()

	validUUID := uuid.New()
	txUUID := uuid.New()

	tests := []struct {
		name      string
		args      args
		setupMock func(svc *mocks.PaymentService)
		expected  expected
	}{
		{
			name: "пустой order_uuid",
			args: args{
				req: &paymentv1.PayOrderRequest{
					OrderUuid: "",
				},
			},
			setupMock: func(svc *mocks.PaymentService) {},
			expected: expected{
				errCode: codes.InvalidArgument,
			},
		},
		{
			name: "невалидный order_uuid",
			args: args{
				req: &paymentv1.PayOrderRequest{
					OrderUuid: "not-a-uuid",
				},
			},
			setupMock: func(svc *mocks.PaymentService) {},
			expected: expected{
				errCode: codes.InvalidArgument,
			},
		},
		{
			name: "ошибка сервиса оплаты",
			args: args{
				req: &paymentv1.PayOrderRequest{
					OrderUuid: validUUID.String(),
				},
			},
			setupMock: func(svc *mocks.PaymentService) {
				svc.EXPECT().
					PayOrder(ctx, validUUID, mock.Anything).
					Return((*uuid.UUID)(nil), errs.ErrPaymentMethodNotSpecified)
			},
			expected: expected{
				err: errs.ErrPaymentMethodNotSpecified,
			},
		},
		{
			name: "успешная оплата заказа",
			args: args{
				req: &paymentv1.PayOrderRequest{
					OrderUuid: validUUID.String(),
				},
			},
			setupMock: func(svc *mocks.PaymentService) {
				svc.EXPECT().
					PayOrder(ctx, validUUID, mock.Anything).
					Return(&txUUID, nil)
			},
			expected: expected{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			paymentService := mocks.NewPaymentService(t)
			api := &api{
				paymentService: paymentService,
			}

			tc.setupMock(paymentService)

			res, err := api.PayOrder(ctx, tc.args.req)

			if tc.expected.errCode != 0 {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, tc.expected.errCode, st.Code())
				return
			}

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.expected.err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, res)
			assert.Equal(t, txUUID.String(), res.TransactionUuid)
		})
	}
}
