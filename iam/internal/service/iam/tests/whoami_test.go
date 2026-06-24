package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/vixart/rocket-factory/iam/internal/model"
	"github.com/vixart/rocket-factory/iam/internal/service/iam"
	"github.com/vixart/rocket-factory/iam/internal/service/iam/mocks"
)

func TestWhoami(t *testing.T) {
	t.Parallel()

	type args struct {
		sessionUUID uuid.UUID
	}

	type expected struct {
		user    model.User
		session model.Session
		err     error
	}

	var (
		ctx         = context.Background()
		sessionUUID = uuid.New()
		userUUID    = uuid.New()
		storageErr  = errors.New("storage error")
		now         = time.Now().UTC().Truncate(time.Second)

		user = model.User{
			UUID:      userUUID,
			Login:     "user@example.com",
			CreatedAt: now,
		}

		session = model.Session{
			UUID:      sessionUUID,
			UserUUID:  userUUID,
			Login:     "user@example.com",
			CreatedAt: now,
			ExpiresAt: now.Add(time.Hour),
		}
	)

	tests := []struct {
		name      string
		args      args
		setupMock func(userRepo *mocks.UserRepository, sessionRepo *mocks.SessionRepository)
		expected  expected
	}{
		{
			name: "успешное получение сессии",
			args: args{
				sessionUUID: sessionUUID,
			},
			setupMock: func(userRepo *mocks.UserRepository, sessionRepo *mocks.SessionRepository) {
				sessionRepo.EXPECT().
					Get(ctx, sessionUUID).
					Return(user, session, nil)
			},
			expected: expected{
				user:    user,
				session: session,
			},
		},
		{
			name: "ошибка хранилища",
			args: args{
				sessionUUID: sessionUUID,
			},
			setupMock: func(userRepo *mocks.UserRepository, sessionRepo *mocks.SessionRepository) {
				sessionRepo.EXPECT().
					Get(ctx, sessionUUID).
					Return(model.User{}, model.Session{}, storageErr)
			},
			expected: expected{
				err: storageErr,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			userRepo := mocks.NewUserRepository(t)
			sessionRepo := mocks.NewSessionRepository(t)
			svc := iam.NewService(userRepo, sessionRepo, time.Hour, bcrypt.MinCost)

			tc.setupMock(userRepo, sessionRepo)

			resultUser, resultSession, err := svc.Whoami(ctx, tc.args.sessionUUID)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.expected.err.Error())
				assert.Equal(t, model.User{}, resultUser)
				assert.Equal(t, model.Session{}, resultSession)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expected.user, resultUser)
			assert.Equal(t, tc.expected.session, resultSession)
		})
	}
}
