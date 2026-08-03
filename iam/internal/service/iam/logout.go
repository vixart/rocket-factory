package iam

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

func (s *service) Logout(ctx context.Context, sessionUUID uuid.UUID) error {
	if err := s.sessionStorage.Delete(ctx, sessionUUID); err != nil {
		return err
	}

	slog.InfoContext(ctx, "сессия завершена")

	return nil
}
