package iam

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/iam/internal/model"
)

type UserRepository interface {
	GetByUUID(ctx context.Context, userUUID uuid.UUID) (model.User, error)
	GetByLogin(ctx context.Context, login string) (model.User, error)
	Create(ctx context.Context, user model.User) error
}

type SessionRepository interface {
	Set(ctx context.Context, sessionUUID uuid.UUID, user model.User, ttl time.Duration) error
	Delete(ctx context.Context, sessionUUID uuid.UUID) error
	Get(ctx context.Context, sessionUuid uuid.UUID) (model.User, model.Session, error)
}
