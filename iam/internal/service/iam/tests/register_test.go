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

func TestRegister(t *testing.T) {
	t.Parallel()

	type args struct {
		input input.UserRegisterInput
	}

	type expected struct {
		err error
	}

	var (
		ctx     = context.Background()
		repoErr = errors.New("repo error")
	)

	tests := []struct {
		name      string
		args      args
		setupMock func(userRepo *mocks.UserRepository, sessionRepo *mocks.SessionRepository)
		expected  expected
	}{
		{
			name: "успешная регистрация",
			args: args{
				input: input.UserRegisterInput{
					Login:    "user@example.com",
					Password: "secret123",
				},
			},
			setupMock: func(userRepo *mocks.UserRepository, sessionRepo *mocks.SessionRepository) {
				userRepo.EXPECT().
					Create(ctx, mock.MatchedBy(func(u model.User) bool {
						return u.Login == "user@example.com" &&
							u.PasswordHash != "" &&
							u.UUID != uuid.Nil
					})).
					Return(nil)
			},
		},
		{
			name: "пустой логин",
			args: args{
				input: input.UserRegisterInput{
					Login:    "",
					Password: "secret123",
				},
			},
			setupMock: func(userRepo *mocks.UserRepository, sessionRepo *mocks.SessionRepository) {},
			expected: expected{
				err: errs.ErrEmptyCredential,
			},
		},
		{
			name: "пустой пароль",
			args: args{
				input: input.UserRegisterInput{
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
			name: "слабый пароль",
			args: args{
				input: input.UserRegisterInput{
					Login:    "user@example.com",
					Password: "short",
				},
			},
			setupMock: func(userRepo *mocks.UserRepository, sessionRepo *mocks.SessionRepository) {},
			expected: expected{
				err: errs.ErrWeakPassword,
			},
		},
		{
			name: "ошибка репозитория при создании",
			args: args{
				input: input.UserRegisterInput{
					Login:    "user@example.com",
					Password: "secret123",
				},
			},
			setupMock: func(userRepo *mocks.UserRepository, sessionRepo *mocks.SessionRepository) {
				userRepo.EXPECT().
					Create(ctx, mock.MatchedBy(func(u model.User) bool {
						return u.Login == "user@example.com"
					})).
					Return(repoErr)
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

			tc.setupMock(userRepo, sessionRepo)

			result, err := svc.Register(ctx, tc.args.input)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.expected.err.Error())
				assert.Equal(t, model.User{}, result)

				return
			}

			require.NoError(t, err)
			assert.NotEqual(t, uuid.Nil, result.UUID)
			assert.Equal(t, tc.args.input.Login, result.Login)
			assert.NotEmpty(t, result.PasswordHash)
			assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(result.PasswordHash), []byte(tc.args.input.Password)))
		})
	}
}
