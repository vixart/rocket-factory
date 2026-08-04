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

	"github.com/vixart/rocket-factory/iam/internal/service/iam"
	"github.com/vixart/rocket-factory/iam/internal/service/iam/mocks"
)

func TestLogout(t *testing.T) {
	t.Parallel()

	type args struct {
		sessionUUID uuid.UUID
	}

	type expected struct {
		err error
	}

	var (
		ctx         = context.Background()
		sessionUUID = uuid.New()
		storageErr  = errors.New("storage error")
	)

	tests := []struct {
		name      string
		args      args
		setupMock func(userRepo *mocks.UserRepository, sessionRepo *mocks.SessionRepository)
		expected  expected
	}{
		{
			name: "logout succeeds",
			args: args{
				sessionUUID: sessionUUID,
			},
			setupMock: func(userRepo *mocks.UserRepository, sessionRepo *mocks.SessionRepository) {
				sessionRepo.EXPECT().
					Delete(ctx, sessionUUID).
					Return(nil)
			},
		},
		{
			name: "storage fails",
			args: args{
				sessionUUID: sessionUUID,
			},
			setupMock: func(userRepo *mocks.UserRepository, sessionRepo *mocks.SessionRepository) {
				sessionRepo.EXPECT().
					Delete(ctx, sessionUUID).
					Return(storageErr)
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

			err := svc.Logout(ctx, tc.args.sessionUUID)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.expected.err.Error())

				return
			}

			require.NoError(t, err)
		})
	}
}
