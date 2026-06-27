package v1

import (
	"context"

	"github.com/vixart/rocket-factory/iam/internal/api/converter"
	userv1 "github.com/vixart/rocket-factory/shared/pkg/proto/user/v1"
)

func (a *api) Register(ctx context.Context, req *userv1.RegisterRequest) (*userv1.RegisterResponse, error) {
	registerInput := converter.RegisterRequestToRegisterInput(req)
	user, err := a.userService.Register(ctx, registerInput)
	if err != nil {
		return nil, err
	}

	return &userv1.RegisterResponse{
		UserUuid: user.UUID.String(),
	}, nil
}
