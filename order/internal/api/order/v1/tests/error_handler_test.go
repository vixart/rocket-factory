package tests

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ogen-go/ogen/ogenerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vixart/rocket-factory/order/internal/api/order/v1"
	errs "github.com/vixart/rocket-factory/order/internal/errors"
)

func TestErrorHandler(t *testing.T) {
	t.Parallel()

	type expected struct {
		statusCode int
		body       string
	}

	tests := []struct {
		name     string
		err      error
		expected expected
	}{
		{
			name: "ошибка декодирования параметров",
			err: &ogenerrors.DecodeParamsError{
				Err: errors.New("invalid uuid"),
			},
			expected: expected{
				statusCode: http.StatusBadRequest,
				body:       `{"code":400,"message":"operation : decode params: invalid uuid"}`,
			},
		},
		{
			name: "некорректный статус заказа",
			err:  errs.ErrInvalidOrderStatus,
			expected: expected{
				statusCode: http.StatusConflict,
				body:       `{"code":409,"message":"заказ имеет недопустимый статус"}`,
			},
		},
		{
			name: "детали нет на складе",
			err:  errs.ErrPartInsufficientStock,
			expected: expected{
				statusCode: http.StatusConflict,
				body:       `{"code":409,"message":"детали нет на складе"}`,
			},
		},
		{
			name: "ошибка оплаты заказа",
			err:  errs.ErrPaymentFailed,
			expected: expected{
				statusCode: http.StatusBadRequest,
				body:       `{"code":400,"message":"не удалось оплатить заказ"}`,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/", nil)

			v1.ErrorHandler(context.Background(), recorder, request, tc.err)

			response := recorder.Result()

			require.Equal(t, tc.expected.statusCode, response.StatusCode)
			assert.JSONEq(t, tc.expected.body, recorder.Body.String())
		})
	}
}
