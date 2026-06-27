package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/vixart/rocket-factory/iam/internal/api/auth/v1"
	"github.com/vixart/rocket-factory/iam/internal/api/auth/v1/mocks"
	"github.com/vixart/rocket-factory/iam/internal/service/input"
	authv1 "github.com/vixart/rocket-factory/shared/pkg/proto/auth/v1"
)

func TestLogin(t *testing.T) {
	t.Parallel()

	type args struct {
		req *authv1.LoginRequest
	}

	type expected struct {
		sessionUUID uuid.UUID
		err         error
	}

	var (
		ctx         = context.Background()
		sessionUUID = uuid.New()
		serviceErr  = errors.New("service error")
	)

	tests := []struct {
		name      string
		args      args
		setupMock func(svc *mocks.SessionService)
		expected  expected
	}{
		{
			name: "успешный логин",
			args: args{
				req: &authv1.LoginRequest{
					Login:    "user@example.com",
					Password: "secret",
				},
			},
			setupMock: func(svc *mocks.SessionService) {
				svc.EXPECT().
					Login(
						ctx,
						input.UserLoginInput{
							Login:    "user@example.com",
							Password: "secret",
						},
					).
					Return(sessionUUID, nil)
			},
			expected: expected{
				sessionUUID: sessionUUID,
			},
		},
		{
			name: "ошибка сервиса",
			args: args{
				req: &authv1.LoginRequest{
					Login:    "user@example.com",
					Password: "wrong",
				},
			},
			setupMock: func(svc *mocks.SessionService) {
				svc.EXPECT().
					Login(
						ctx,
						input.UserLoginInput{
							Login:    "user@example.com",
							Password: "wrong",
						},
					).
					Return(uuid.UUID{}, serviceErr)
			},
			expected: expected{
				err: serviceErr,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockSvc := mocks.NewSessionService(t)
			api := v1.NewApi(mockSvc)

			tc.setupMock(mockSvc)

			res, err := api.Login(ctx, tc.args.req)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.expected.err.Error())
				assert.Nil(t, res)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, res)
			assert.Equal(t, tc.expected.sessionUUID.String(), res.SessionUuid)
		})
	}
}
