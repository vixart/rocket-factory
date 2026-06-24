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
	errs "github.com/vixart/rocket-factory/iam/internal/errors"
	authv1 "github.com/vixart/rocket-factory/shared/pkg/proto/auth/v1"
)

func TestLogout(t *testing.T) {
	t.Parallel()

	type args struct {
		req *authv1.LogoutRequest
	}

	type expected struct {
		err error
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
			name: "успешный логаут",
			args: args{
				req: &authv1.LogoutRequest{
					SessionUuid: sessionUUID.String(),
				},
			},
			setupMock: func(svc *mocks.SessionService) {
				svc.EXPECT().
					Logout(ctx, sessionUUID).
					Return(nil)
			},
		},
		{
			name: "пустой session uuid",
			args: args{
				req: &authv1.LogoutRequest{
					SessionUuid: "",
				},
			},
			setupMock: func(svc *mocks.SessionService) {},
			expected: expected{
				err: errs.ErrEmptySessionID,
			},
		},
		{
			name: "невалидный session uuid",
			args: args{
				req: &authv1.LogoutRequest{
					SessionUuid: "not-a-uuid",
				},
			},
			setupMock: func(svc *mocks.SessionService) {},
			expected: expected{
				err: errs.ErrInvalidUUID,
			},
		},
		{
			name: "ошибка сервиса",
			args: args{
				req: &authv1.LogoutRequest{
					SessionUuid: sessionUUID.String(),
				},
			},
			setupMock: func(svc *mocks.SessionService) {
				svc.EXPECT().
					Logout(ctx, sessionUUID).
					Return(serviceErr)
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

			res, err := api.Logout(ctx, tc.args.req)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.expected.err.Error())
				assert.Nil(t, res)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, res)
		})
	}
}
