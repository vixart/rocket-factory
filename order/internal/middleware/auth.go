package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/platform/pkg/auth"
)

type IAMService interface {
	Whoami(ctx context.Context, sessionUUID uuid.UUID) (uuid.UUID, uuid.UUID, error)
}

type authMiddleware struct {
	iamService IAMService
}

func NewAuthMiddleware(iamService IAMService) *authMiddleware {
	return &authMiddleware{
		iamService: iamService,
	}
}

func (m *authMiddleware) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header is missing", http.StatusUnauthorized)
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "expected format: Bearer <token>", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			http.Error(w, "token is empty", http.StatusUnauthorized)
			return
		}

		sessionUUID, err := uuid.Parse(token)
		if err != nil {
			http.Error(w, "token has an invalid format", http.StatusUnauthorized)
			return
		}

		userUUID, sessionUUID, err := m.iamService.Whoami(r.Context(), sessionUUID)
		if err != nil {
			http.Error(w, "session is not valid", http.StatusUnauthorized)
			return
		}

		ctx := auth.WithUserUUID(r.Context(), userUUID)
		ctx = auth.WithSessionUUID(ctx, sessionUUID)

		slog.Debug(
			"session established",
			slog.String("userUUID", userUUID.String()),
			slog.String("sessionUUID", sessionUUID.String()),
		)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
