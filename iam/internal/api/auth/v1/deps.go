package v1

import (
	"context"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/iam/internal/model"
	"github.com/vixart/rocket-factory/iam/internal/service/input"
)

type SessionService interface {
	Login(ctx context.Context, input input.UserLoginInput) (uuid.UUID, error)
	Logout(ctx context.Context, sessionUUID uuid.UUID) error
	Whoami(ctx context.Context, sessionUUID uuid.UUID) (model.User, model.Session, error)
}
