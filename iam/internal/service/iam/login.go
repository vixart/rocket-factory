package iam

import (
	"context"

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
		return uuid.Nil, errs.ErrInvalidCredentials
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password))
	if err != nil {
		return uuid.Nil, errs.ErrInvalidCredentials
	}

	sessionUUID := uuid.New()

	err = s.sessionStorage.Set(ctx, sessionUUID, user, s.sessionTTL)
	if err != nil {
		return uuid.Nil, err
	}

	return sessionUUID, nil
}
