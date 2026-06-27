package iam

import (
	"context"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/iam/internal/model"
)

func (s *service) GetUser(ctx context.Context, userUuid uuid.UUID) (model.User, error) {
	user, err := s.userRepository.GetByUUID(ctx, userUuid)
	if err != nil {
		return model.User{}, err
	}

	return user, nil
}
