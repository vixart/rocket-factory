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
			name: "parameter decoding error",
			err: &ogenerrors.DecodeParamsError{
				Err: errors.New("invalid uuid"),
			},
			expected: expected{
				statusCode: http.StatusBadRequest,
				body:       `{"code":400,"message":"operation : decode params: invalid uuid"}`,
			},
		},
		{
			name: "request body decoding error",
			err: &ogenerrors.DecodeRequestError{
				Err: errors.New("invalid body"),
			},
			expected: expected{
				statusCode: http.StatusBadRequest,
				body:       `{"code":400,"message":"operation : decode request: invalid body"}`,
			},
		},
		{
			name: "order not found",
			err:  errs.ErrOrderNotFound,
			expected: expected{
				statusCode: http.StatusNotFound,
				body:       `{"code":404,"message":"` + errs.ErrOrderNotFound.Error() + `"}`,
			},
		},
		{
			name: "part not found",
			err:  errs.ErrPartNotFound,
			expected: expected{
				statusCode: http.StatusNotFound,
				body:       `{"code":404,"message":"` + errs.ErrPartNotFound.Error() + `"}`,
			},
		},
		{
			name: "unauthorized",
			err:  errs.ErrUnauthorized,
			expected: expected{
				statusCode: http.StatusUnauthorized,
				body:       `{"code":401,"message":"` + errs.ErrUnauthorized.Error() + `"}`,
			},
		},
		{
			name: "invalid order status",
			err:  errs.ErrInvalidOrderStatus,
			expected: expected{
				statusCode: http.StatusConflict,
				body:       `{"code":409,"message":"` + errs.ErrInvalidOrderStatus.Error() + `"}`,
			},
		},
		{
			name: "incompatible parts",
			err:  errs.ErrIncompatibleParts,
			expected: expected{
				statusCode: http.StatusConflict,
				body:       `{"code":409,"message":"` + errs.ErrIncompatibleParts.Error() + `"}`,
			},
		},
		{
			name: "part is out of stock",
			err:  errs.ErrOutOfStock,
			expected: expected{
				statusCode: http.StatusConflict,
				body:       `{"code":409,"message":"` + errs.ErrOutOfStock.Error() + `"}`,
			},
		},
		{
			name: "invalid uuid",
			err:  errs.ErrInvalidUUID,
			expected: expected{
				statusCode: http.StatusBadRequest,
				body:       `{"code":400,"message":"` + errs.ErrInvalidUUID.Error() + `"}`,
			},
		},
		{
			name: "order payment fails",
			err:  errs.ErrPaymentFailed,
			expected: expected{
				statusCode: http.StatusBadRequest,
				body:       `{"code":400,"message":"` + errs.ErrPaymentFailed.Error() + `"}`,
			},
		},
		{
			name: "part type mismatch",
			err:  errs.ErrPartTypeMismatch,
			expected: expected{
				statusCode: http.StatusBadRequest,
				body:       `{"code":400,"message":"` + errs.ErrPartTypeMismatch.Error() + `"}`,
			},
		},
		{
			name: "invalid payment method",
			err:  errs.ErrInvalidPaymentMethod,
			expected: expected{
				statusCode: http.StatusBadRequest,
				body:       `{"code":400,"message":"` + errs.ErrInvalidPaymentMethod.Error() + `"}`,
			},
		},
		{
			name: "unknown error",
			err:  errors.New("boom"),
			expected: expected{
				statusCode: http.StatusInternalServerError,
				body:       `{"code":500,"message":"internal error"}`,
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
