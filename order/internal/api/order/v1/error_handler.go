package v1

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/ogen-go/ogen/ogenerrors"

	errs "github.com/vixart/rocket-factory/order/internal/errors"
)

type errorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ErrorHandler is the global ogen hook, wired in via orderv1.WithErrorHandler.
func ErrorHandler(ctx context.Context, w http.ResponseWriter, _ *http.Request, err error) {
	code, message := mapError(err)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	if encErr := json.NewEncoder(w).Encode(errorResponse{Code: code, Message: message}); encErr != nil {
		slog.ErrorContext(ctx, "failed to encode the response", "error", encErr)
	}
}

func mapError(err error) (int, string) {
	// Request decoding and validation errors from ogen are always 400.
	var decodeParams *ogenerrors.DecodeParamsError
	var decodeRequest *ogenerrors.DecodeRequestError

	switch {
	case errors.As(err, &decodeParams), errors.As(err, &decodeRequest):
		return http.StatusBadRequest, err.Error()

	// 404 Not Found
	case errors.Is(err, errs.ErrOrderNotFound),
		errors.Is(err, errs.ErrPartNotFound):
		return http.StatusNotFound, err.Error()

	// 401 Unauthorized
	case errors.Is(err, errs.ErrUnauthorized):
		return http.StatusUnauthorized, err.Error()

	// 409 Conflict
	case errors.Is(err, errs.ErrInvalidOrderStatus),
		errors.Is(err, errs.ErrIncompatibleParts),
		errors.Is(err, errs.ErrOutOfStock):
		return http.StatusConflict, err.Error()

	// 400 Bad Request
	case errors.Is(err, errs.ErrInvalidUUID),
		errors.Is(err, errs.ErrPaymentFailed),
		errors.Is(err, errs.ErrPartTypeMismatch),
		errors.Is(err, errs.ErrInvalidPaymentMethod):
		return http.StatusBadRequest, err.Error()

	// 500 Internal Server Error
	default:
		return http.StatusInternalServerError, "internal error"
	}
}
