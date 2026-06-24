package iam

import (
	"context"
	"time"

	"github.com/go-faster/errors"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	errs "github.com/vixart/rocket-factory/iam/internal/errors"
	"github.com/vixart/rocket-factory/iam/internal/model"
	"github.com/vixart/rocket-factory/iam/internal/service/input"
)

func (s *service) Register(ctx context.Context, input input.UserRegisterInput) (model.User, error) {
	if err := validateData(input.Login, input.Password); err != nil {
		return model.User{}, err
	}

	_, err := s.userRepository.GetByLogin(ctx, input.Login)

	if err == nil {
		return model.User{}, errs.ErrUserAlreadyExists
	}

	if !errors.Is(err, errs.ErrUserNotFound) {
		return model.User{}, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), s.bcryptCost)
	if err != nil {
		return model.User{}, err
	}

	user := model.User{
		UUID:         uuid.New(),
		Login:        input.Login,
		PasswordHash: string(hash),
		CreatedAt:    time.Now(),
		UpdatedAt:    nil,
	}

	err = s.userRepository.Create(ctx, user)
	if err != nil {
		return model.User{}, err
	}

	return user, nil
}

func validateData(login, password string) error {
	if login == "" || password == "" {
		return errs.ErrEmptyCredential
	}

	if len(password) < 8 {
		return errs.ErrWeakPassword
	}

	return nil
}
