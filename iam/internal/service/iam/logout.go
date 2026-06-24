package iam

import (
	"context"

	"github.com/google/uuid"
)

func (s *service) Logout(ctx context.Context, sessionUUID uuid.UUID) error {
	return s.sessionStorage.Delete(ctx, sessionUUID)
}
