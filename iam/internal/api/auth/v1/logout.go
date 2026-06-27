package v1

import (
	"context"

	"github.com/google/uuid"

	errs "github.com/vixart/rocket-factory/iam/internal/errors"
	authv1 "github.com/vixart/rocket-factory/shared/pkg/proto/auth/v1"
)

func (a *api) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	if req.GetSessionUuid() == "" {
		return nil, errs.ErrEmptySessionID
	}

	sessionUUID, err := uuid.Parse(req.GetSessionUuid())
	if err != nil {
		return nil, errs.ErrInvalidUUID
	}

	err = a.sessionService.Logout(ctx, sessionUUID)
	if err != nil {
		return nil, err
	}

	return &authv1.LogoutResponse{}, nil
}
