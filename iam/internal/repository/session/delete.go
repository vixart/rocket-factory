package session

import (
	"context"

	"github.com/google/uuid"
)

func (r *repository) Delete(ctx context.Context, sessionUUID uuid.UUID) error {
	sessionKey := r.getSessionKey(sessionUUID.String())
	return r.client.Del(ctx, sessionKey).Err()
}
