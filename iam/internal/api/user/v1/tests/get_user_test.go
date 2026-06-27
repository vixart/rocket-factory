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
	errs "github.com/vixart/rocket-factory/iam/internal/errors"
	"github.com/vixart/rocket-factory/iam/internal/model"
	userv1 "github.com/vixart/rocket-factory/shared/pkg/proto/user/v1"
)

func TestGetUser(t *testing.T) {
	t.Parallel()

	type args struct {
		req *userv1.GetUserRequest
	}

	type expected struct {
		err error
	}

	var (
		ctx        = context.Background()
		userUUID   = uuid.New()
		serviceErr = errors.New("service error")
		now        = time.Now().UTC().Truncate(time.Second)
		updatedAt  = now.Add(time.Hour)

		user = model.User{
			UUID:      userUUID,
			Login:     "user@example.com",
			CreatedAt: now,
		}

		userWithUpdatedAt = model.User{
			UUID:      userUUID,
			Login:     "user@example.com",
			CreatedAt: now,
			UpdatedAt: &updatedAt,
		}
	)

	tests := []struct {
		name      string
		args      args
		setupMock func(svc *mocks.UserService)
		expected  expected
	}{
		{
			name: "успешное получение пользователя",
			args: args{
				req: &userv1.GetUserRequest{
					UserUuid: userUUID.String(),
				},
			},
			setupMock: func(svc *mocks.UserService) {
				svc.EXPECT().
					GetUser(ctx, userUUID).
					Return(user, nil)
			},
		},
		{
			name: "успешное получение пользователя с updatedAt",
			args: args{
				req: &userv1.GetUserRequest{
					UserUuid: userUUID.String(),
				},
			},
			setupMock: func(svc *mocks.UserService) {
				svc.EXPECT().
					GetUser(ctx, userUUID).
					Return(userWithUpdatedAt, nil)
			},
		},
		{
			name: "невалидный uuid",
			args: args{
				req: &userv1.GetUserRequest{
					UserUuid: "not-a-uuid",
				},
			},
			setupMock: func(svc *mocks.UserService) {},
			expected: expected{
				err: errs.ErrInvalidUUID,
			},
		},
		{
			name: "ошибка сервиса",
			args: args{
				req: &userv1.GetUserRequest{
					UserUuid: userUUID.String(),
				},
			},
			setupMock: func(svc *mocks.UserService) {
				svc.EXPECT().
					GetUser(ctx, userUUID).
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

			res, err := api.GetUser(ctx, tc.args.req)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.expected.err.Error())
				assert.Nil(t, res)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, res)
			require.NotNil(t, res.User)
			require.NotNil(t, res.User.Info)

			assert.Equal(t, userUUID.String(), res.User.Uuid)
			assert.Equal(t, user.Login, res.User.Info.Login)
			assert.Equal(t, now.Unix(), res.User.CreatedAt.AsTime().Unix())
		})
	}
}
