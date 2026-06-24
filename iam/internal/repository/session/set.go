package session

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/iam/internal/model"
	"github.com/vixart/rocket-factory/iam/internal/repository/converter"
)

func (r *repository) Set(ctx context.Context, sessionUUID uuid.UUID, user model.User, ttl time.Duration) error {
	sessionKey := r.getSessionKey(sessionUUID.String())
	expiresAt := time.Now().Add(ttl)
	err := r.client.HSet(ctx, sessionKey, converter.UserModelToSessionView(sessionUUID, user, expiresAt)).Err()
	if err != nil {
		return err
	}

	return r.client.ExpireAt(ctx, sessionKey, expiresAt).Err()
}
