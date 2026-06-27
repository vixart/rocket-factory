package v1

import (
	"context"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/iam/internal/model"
	"github.com/vixart/rocket-factory/iam/internal/service/input"
)

type UserService interface {
	GetUser(ctx context.Context, userUuid uuid.UUID) (model.User, error)
	Register(ctx context.Context, input input.UserRegisterInput) (model.User, error)
}
