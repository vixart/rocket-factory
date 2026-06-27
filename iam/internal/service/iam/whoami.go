package iam

import (
	"context"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/iam/internal/model"
)

func (s *service) Whoami(ctx context.Context, sessionUUID uuid.UUID) (model.User, model.Session, error) {
	return s.sessionStorage.Get(ctx, sessionUUID)
}
