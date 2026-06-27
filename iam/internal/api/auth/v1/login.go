package v1

import (
	"context"

	"github.com/vixart/rocket-factory/iam/internal/api/converter"
	authv1 "github.com/vixart/rocket-factory/shared/pkg/proto/auth/v1"
)

func (a *api) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	loginInput := converter.LoginRequestToLoginInput(req)
	sessionUUID, err := a.sessionService.Login(ctx, loginInput)
	if err != nil {
		return nil, err
	}

	return &authv1.LoginResponse{
		SessionUuid: sessionUUID.String(),
	}, nil
}
