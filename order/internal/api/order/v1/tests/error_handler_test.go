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

	v1 "github.com/vixart/rocket-factory/order/internal/api/order/v1"
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
			name: "заказ не найден",
			err:  errs.ErrOrderNotFound,
			expected: expected{
				statusCode: http.StatusNotFound,
				body:       `{"code":404,"message":"заказ не найден"}`,
			},
		},
		{
			name: "деталь не найдена",
			err:  errs.ErrPartNotFound,
			expected: expected{
				statusCode: http.StatusNotFound,
				body:       `{"code":404,"message":"деталь не найдена"}`,
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
			name: "несовместимые детали",
			err:  errs.ErrIncompatibleParts,
			expected: expected{
				statusCode: http.StatusConflict,
				body:       `{"code":409,"message":"детали несовместимы"}`,
			},
		},
		{
			name: "детали нет на складе",
			err:  errs.ErrOutOfStock,
			expected: expected{
				statusCode: http.StatusConflict,
				body:       `{"code":409,"message":"детали нет на складе"}`,
			},
		},
		{
			name: "некорректный uuid",
			err:  errs.ErrInvalidUUID,
			expected: expected{
				statusCode: http.StatusBadRequest,
				body:       `{"code":400,"message":"некорректный uuid"}`,
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
		{
			name: "некорректный способ оплаты",
			err:  errs.ErrInvalidPaymentMethod,
			expected: expected{
				statusCode: http.StatusBadRequest,
				body:       `{"code":400,"message":"некорректный способ оплаты"}`,
			},
		},
		{
			name: "неизвестная ошибка",
			err:  errors.New("boom"),
			expected: expected{
				statusCode: http.StatusInternalServerError,
				body:       `{"code":500,"message":"внутренняя ошибка"}`,
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
			assert.Equal(t, "application/json", response.Header.Get("Content-Type"))
			assert.JSONEq(t, tc.expected.body, recorder.Body.String())
		})
	}
}
