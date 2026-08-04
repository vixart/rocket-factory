package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	errs "github.com/vixart/rocket-factory/iam/internal/errors"
	"github.com/vixart/rocket-factory/iam/internal/model"
	"github.com/vixart/rocket-factory/iam/internal/service/iam"
	"github.com/vixart/rocket-factory/iam/internal/service/iam/mocks"
	"github.com/vixart/rocket-factory/iam/internal/service/input"
)

func TestLogin(t *testing.T) {
	t.Parallel()

	type args struct {
		input input.UserLoginInput
	}

	type expected struct {
		err error
	}

	var (
		ctx        = context.Background()
		repoErr    = errors.New("repo error")
		storageErr = errors.New("storage error")
		now        = time.Now().UTC().Truncate(time.Second)

		passwordHash, _ = bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)

		user = model.User{
			UUID:         uuid.New(),
			Login:        "user@example.com",
			PasswordHash: string(passwordHash),
			CreatedAt:    now,
		}
	)

	tests := []struct {
		name      string
		args      args
		setupMock func(userRepo *mocks.UserRepository, sessionRepo *mocks.SessionRepository)
		expected  expected
	}{
		{
			name: "login succeeds",
			args: args{
				input: input.UserLoginInput{
					Login:    "user@example.com",
					Password: "secret",
				},
			},
			setupMock: func(userRepo *mocks.UserRepository, sessionRepo *mocks.SessionRepository) {
				userRepo.EXPECT().
					GetByLogin(ctx, "user@example.com").
					Return(user, nil)
				sessionRepo.EXPECT().
					Set(ctx, mock.AnythingOfType("uuid.UUID"), user, time.Hour).
					Return(nil)
			},
		},
		{
			name: "empty login",
			args: args{
				input: input.UserLoginInput{
					Login:    "",
					Password: "secret",
				},
			},
			setupMock: func(userRepo *mocks.UserRepository, sessionRepo *mocks.SessionRepository) {},
			expected: expected{
				err: errs.ErrEmptyCredential,
			},
		},
		{
			name: "empty password",
			args: args{
				input: input.UserLoginInput{
					Login:    "user@example.com",
					Password: "",
				},
			},
			setupMock: func(userRepo *mocks.UserRepository, sessionRepo *mocks.SessionRepository) {},
			expected: expected{
				err: errs.ErrEmptyCredential,
			},
		},
		{
			name: "user not found",
			args: args{
				input: input.UserLoginInput{
					Login:    "user@example.com",
					Password: "secret",
				},
			},
			setupMock: func(userRepo *mocks.UserRepository, sessionRepo *mocks.SessionRepository) {
				userRepo.EXPECT().
					GetByLogin(ctx, "user@example.com").
					Return(model.User{}, repoErr)
			},
			expected: expected{
				err: errs.ErrInvalidCredentials,
			},
		},
		{
			name: "wrong password",
			args: args{
				input: input.UserLoginInput{
					Login:    "user@example.com",
					Password: "wrong",
				},
			},
			setupMock: func(userRepo *mocks.UserRepository, sessionRepo *mocks.SessionRepository) {
				userRepo.EXPECT().
					GetByLogin(ctx, "user@example.com").
					Return(user, nil)
			},
			expected: expected{
				err: errs.ErrInvalidCredentials,
			},
		},
		{
			name: "session storage fails",
			args: args{
				input: input.UserLoginInput{
					Login:    "user@example.com",
					Password: "secret",
				},
			},
			setupMock: func(userRepo *mocks.UserRepository, sessionRepo *mocks.SessionRepository) {
				userRepo.EXPECT().
					GetByLogin(ctx, "user@example.com").
					Return(user, nil)
				sessionRepo.EXPECT().
					Set(ctx, mock.AnythingOfType("uuid.UUID"), user, time.Hour).
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

			result, err := svc.Login(ctx, tc.args.input)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.expected.err.Error())
				assert.Equal(t, uuid.Nil, result)

				return
			}

			require.NoError(t, err)
			assert.NotEqual(t, uuid.Nil, result)
		})
	}
}
