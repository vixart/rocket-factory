package tests

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	errs "github.com/vixart/rocket-factory/payment/internal/errors"
	"github.com/vixart/rocket-factory/payment/internal/model"
	"github.com/vixart/rocket-factory/payment/internal/service/payment"
)

func TestPayOrder(t *testing.T) {
	t.Parallel()

	type args struct {
		orderID       uuid.UUID
		paymentMethod model.PaymentMethod
	}

	type expected struct {
		hasErr bool
		err    error
	}

	tests := []struct {
		name     string
		args     args
		expected expected
	}{
		{
			name: "payment succeeds",
			args: args{
				orderID:       uuid.New(),
				paymentMethod: model.PaymentMethodCard,
			},
			expected: expected{
				hasErr: false,
			},
		},
		{
			name: "payment method is not specified",
			args: args{
				orderID:       uuid.New(),
				paymentMethod: model.PaymentMethodUnspecified,
			},
			expected: expected{
				hasErr: true,
				err:    errs.ErrPaymentMethodNotSpecified,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := payment.NewService()

			res, err := svc.PayOrder(t.Context(), tc.args.orderID, tc.args.paymentMethod)

			if tc.expected.hasErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.expected.err)
				assert.Nil(t, res)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, res)
			assert.NotEqual(t, uuid.Nil, *res)
		})
	}
}
