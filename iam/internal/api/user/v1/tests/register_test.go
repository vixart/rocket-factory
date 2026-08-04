package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/vixart/rocket-factory/iam/internal/api/user/v1"
	"github.com/vixart/rocket-factory/iam/internal/api/user/v1/mocks"
	"github.com/vixart/rocket-factory/iam/internal/model"
	"github.com/vixart/rocket-factory/iam/internal/service/input"
	commonv1 "github.com/vixart/rocket-factory/shared/pkg/proto/common/v1"
	userv1 "github.com/vixart/rocket-factory/shared/pkg/proto/user/v1"
)

func TestRegister(t *testing.T) {
	t.Parallel()

	type args struct {
		req *userv1.RegisterRequest
	}

	type expected struct {
		err error
	}

	var (
		ctx        = context.Background()
		userUUID   = uuid.New()
		serviceErr = errors.New("service error")
		now        = time.Now().UTC().Truncate(time.Second)

		user = model.User{
			UUID:      userUUID,
			Login:     "user@example.com",
			CreatedAt: now,
		}
	)

	tests := []struct {
		name      string
		args      args
		setupMock func(svc *mocks.UserService)
		expected  expected
	}{
		{
			name: "registration succeeds",
			args: args{
				req: &userv1.RegisterRequest{
					Info: &userv1.UserRegistrationInfo{
						Info: &commonv1.UserInfo{
							Login: "user@example.com",
						},
						Password: "secret",
					},
				},
			},
			setupMock: func(svc *mocks.UserService) {
				svc.EXPECT().
					Register(
						ctx,
						input.UserRegisterInput{
							Login:    "user@example.com",
							Password: "secret",
						},
					).
					Return(user, nil)
			},
		},
		{
			name: "service fails",
			args: args{
				req: &userv1.RegisterRequest{
					Info: &userv1.UserRegistrationInfo{
						Info: &commonv1.UserInfo{
							Login: "user@example.com",
						},
						Password: "secret",
					},
				},
			},
			setupMock: func(svc *mocks.UserService) {
				svc.EXPECT().
					Register(
						ctx,
						input.UserRegisterInput{
							Login:    "user@example.com",
							Password: "secret",
						},
					).
					Return(model.User{}, serviceErr)
			},
			expected: expected{
				err: serviceErr,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockSvc := mocks.NewUserService(t)
			api := v1.NewApi(mockSvc)

			tc.setupMock(mockSvc)

			res, err := api.Register(ctx, tc.args.req)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.expected.err.Error())
				assert.Nil(t, res)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, res)
			assert.Equal(t, userUUID.String(), res.UserUuid)
		})
	}
}
