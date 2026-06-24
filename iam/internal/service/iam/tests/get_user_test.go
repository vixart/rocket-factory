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

func TestGetUser(t *testing.T) {
	t.Parallel()

	type args struct {
		userUUID uuid.UUID
	}

	type expected struct {
		user model.User
		err  error
	}

	var (
		ctx      = context.Background()
		userUUID = uuid.New()
		repoErr  = errors.New("repo error")
		now      = time.Now().UTC().Truncate(time.Second)

		user = model.User{
			UUID:      userUUID,
			Login:     "user@example.com",
			CreatedAt: now,
		}
	)

	tests := []struct {
		name      string
		args      args
		setupMock func(repo *mocks.UserRepository)
		expected  expected
	}{
		{
			name: "успешное получение пользователя",
			args: args{
				userUUID: userUUID,
			},
			setupMock: func(repo *mocks.UserRepository) {
				repo.EXPECT().
					GetByUUID(ctx, userUUID).
					Return(user, nil)
			},
			expected: expected{
				user: user,
			},
		},
		{
			name: "ошибка репозитория",
			args: args{
				userUUID: userUUID,
			},
			setupMock: func(repo *mocks.UserRepository) {
				repo.EXPECT().
					GetByUUID(ctx, userUUID).
					Return(model.User{}, repoErr)
			},
			expected: expected{
				err: repoErr,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			userRepo := mocks.NewUserRepository(t)
			sessionRepo := mocks.NewSessionRepository(t)
			svc := iam.NewService(userRepo, sessionRepo, time.Hour, bcrypt.MinCost)

			tc.setupMock(userRepo)

			result, err := svc.GetUser(ctx, tc.args.userUUID)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.expected.err.Error())
				assert.Equal(t, model.User{}, result)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expected.user, result)
		})
	}
}
