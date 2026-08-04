package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/vixart/rocket-factory/iam/internal/api/auth/v1"
	"github.com/vixart/rocket-factory/iam/internal/api/auth/v1/mocks"
	errs "github.com/vixart/rocket-factory/iam/internal/errors"
	"github.com/vixart/rocket-factory/iam/internal/model"
	authv1 "github.com/vixart/rocket-factory/shared/pkg/proto/auth/v1"
)

func TestWhoami(t *testing.T) {
	t.Parallel()

	type args struct {
		req *authv1.WhoamiRequest
	}

	type expected struct {
		err error
	}

	var (
		ctx         = context.Background()
		sessionUUID = uuid.New()
		userUUID    = uuid.New()
		serviceErr  = errors.New("service error")
		now         = time.Now().UTC().Truncate(time.Second)
		updatedAt   = now.Add(time.Hour)

		user = model.User{
			UUID:         userUUID,
			Login:        "user@example.com",
			PasswordHash: "hash",
			CreatedAt:    now,
		}

		userWithUpdatedAt = model.User{
			UUID:         userUUID,
			Login:        "user@example.com",
			PasswordHash: "hash",
			CreatedAt:    now,
			UpdatedAt:    &updatedAt,
		}

		session = model.Session{
			UUID:      sessionUUID,
			UserUUID:  userUUID,
			Login:     "user@example.com",
			CreatedAt: now,
			ExpiresAt: now.Add(24 * time.Hour),
		}

		sessionWithUpdatedAt = model.Session{
			UUID:      sessionUUID,
			UserUUID:  userUUID,
			Login:     "user@example.com",
			CreatedAt: now,
			UpdatedAt: &updatedAt,
			ExpiresAt: now.Add(24 * time.Hour),
		}
	)

	tests := []struct {
		name      string
		args      args
		setupMock func(svc *mocks.SessionService)
		expected  expected
	}{
		{
			name: "whoami succeeds",
			args: args{
				req: &authv1.WhoamiRequest{
					SessionUuid: sessionUUID.String(),
				},
			},
			setupMock: func(svc *mocks.SessionService) {
				svc.EXPECT().
					Whoami(ctx, sessionUUID).
					Return(user, session, nil)
			},
		},
		{
			name: "whoami succeeds with updatedAt",
			args: args{
				req: &authv1.WhoamiRequest{
					SessionUuid: sessionUUID.String(),
				},
			},
			setupMock: func(svc *mocks.SessionService) {
				svc.EXPECT().
					Whoami(ctx, sessionUUID).
					Return(userWithUpdatedAt, sessionWithUpdatedAt, nil)
			},
		},
		{
			name: "empty session uuid",
			args: args{
				req: &authv1.WhoamiRequest{
					SessionUuid: "",
				},
			},
			setupMock: func(svc *mocks.SessionService) {},
			expected: expected{
				err: errs.ErrEmptySessionID,
			},
		},
		{
			name: "invalid session uuid",
			args: args{
				req: &authv1.WhoamiRequest{
					SessionUuid: "not-a-uuid",
				},
			},
			setupMock: func(svc *mocks.SessionService) {},
			expected: expected{
				err: errs.ErrInvalidUUID,
			},
		},
		{
			name: "service fails",
			args: args{
				req: &authv1.WhoamiRequest{
					SessionUuid: sessionUUID.String(),
				},
			},
			setupMock: func(svc *mocks.SessionService) {
				svc.EXPECT().
					Whoami(ctx, sessionUUID).
					Return(model.User{}, model.Session{}, serviceErr)
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

			res, err := api.Whoami(ctx, tc.args.req)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.expected.err.Error())
				assert.Nil(t, res)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, res)
			require.NotNil(t, res.Session)
			require.NotNil(t, res.User)

			assert.Equal(t, sessionUUID.String(), res.Session.Uuid)
			assert.Equal(t, now.Unix(), res.Session.CreatedAt.AsTime().Unix())
			assert.Equal(t, session.ExpiresAt.Unix(), res.Session.ExpiresAt.AsTime().Unix())

			assert.Equal(t, userUUID.String(), res.User.Uuid)
			assert.Equal(t, user.Login, res.User.Info.Login)
			assert.Equal(t, now.Unix(), res.User.CreatedAt.AsTime().Unix())
		})
	}
}
