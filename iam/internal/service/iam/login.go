package iam

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	errs "github.com/vixart/rocket-factory/iam/internal/errors"
	"github.com/vixart/rocket-factory/iam/internal/service/input"
)

func (s *service) Login(ctx context.Context, input input.UserLoginInput) (uuid.UUID, error) {
	if input.Login == "" || input.Password == "" {
		return uuid.Nil, errs.ErrEmptyCredential
	}

	user, err := s.userRepository.GetByLogin(ctx, input.Login)
	if err != nil {
		slog.WarnContext(ctx, "вход отклонён: пользователь не найден", "login", input.Login)
		return uuid.Nil, errs.ErrInvalidCredentials
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password))
	if err != nil {
		slog.WarnContext(ctx, "вход отклонён: неверный пароль",
			"login", input.Login, "user_uuid", user.UUID)
		return uuid.Nil, errs.ErrInvalidCredentials
	}

	sessionUUID := uuid.New()

	err = s.sessionStorage.Set(ctx, sessionUUID, user, s.sessionTTL)
	if err != nil {
		return uuid.Nil, err
	}

	slog.InfoContext(ctx, "сессия создана",
		"user_uuid", user.UUID, "login", user.Login, "ttl", s.sessionTTL)

	return sessionUUID, nil
}
