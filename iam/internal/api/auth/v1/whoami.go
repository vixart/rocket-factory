package v1

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	errs "github.com/vixart/rocket-factory/iam/internal/errors"
	"github.com/vixart/rocket-factory/iam/internal/model"
	authv1 "github.com/vixart/rocket-factory/shared/pkg/proto/auth/v1"
	commonv1 "github.com/vixart/rocket-factory/shared/pkg/proto/common/v1"
)

func (a *api) Whoami(ctx context.Context, req *authv1.WhoamiRequest) (*authv1.WhoamiResponse, error) {
	if req.GetSessionUuid() == "" {
		return nil, errs.ErrEmptySessionID
	}

	sessionUUID, err := uuid.Parse(req.GetSessionUuid())
	if err != nil {
		return nil, errs.ErrInvalidUUID
	}

	user, session, err := a.sessionService.Whoami(ctx, sessionUUID)
	if err != nil {
		return nil, err
	}

	return &authv1.WhoamiResponse{
		Session: composeSessionRes(session),
		User:    composeUserRes(user),
	}, nil
}

func composeSessionRes(session model.Session) *commonv1.Session {
	sessionRes := commonv1.Session{
		Uuid:      session.UUID.String(),
		CreatedAt: timestamppb.New(session.CreatedAt),
		ExpiresAt: timestamppb.New(session.ExpiresAt),
	}
	if session.UpdatedAt != nil {
		sessionRes.UpdatedAt = timestamppb.New(*session.UpdatedAt)
	}

	return &sessionRes
}

func composeUserRes(user model.User) *commonv1.User {
	userRes := commonv1.User{
		Uuid: user.UUID.String(),
		Info: &commonv1.UserInfo{
			Login: user.Login,
		},
		CreatedAt: timestamppb.New(user.CreatedAt),
	}
	if user.UpdatedAt != nil {
		userRes.UpdatedAt = timestamppb.New(*user.UpdatedAt)
	}

	return &userRes
}
