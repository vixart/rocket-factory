package v1

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	errs "github.com/vixart/rocket-factory/iam/internal/errors"
	commonv1 "github.com/vixart/rocket-factory/shared/pkg/proto/common/v1"
	userv1 "github.com/vixart/rocket-factory/shared/pkg/proto/user/v1"
)

func (a *api) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	userUuid, err := uuid.Parse(req.GetUserUuid())
	if err != nil {
		return nil, errs.ErrInvalidUUID
	}

	user, err := a.userService.GetUser(ctx, userUuid)
	if err != nil {
		return nil, err
	}

	resp := &userv1.GetUserResponse{
		User: &commonv1.User{
			Uuid: user.UUID.String(),
			Info: &commonv1.UserInfo{
				Login: user.Login,
			},
			CreatedAt: timestamppb.New(user.CreatedAt),
		},
	}

	if user.UpdatedAt != nil {
		resp.User.UpdatedAt = timestamppb.New(*user.UpdatedAt)
	}

	return resp, nil
}
