package session

import (
	"context"

	"github.com/go-faster/errors"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	errs "github.com/vixart/rocket-factory/iam/internal/errors"
	"github.com/vixart/rocket-factory/iam/internal/model"
	"github.com/vixart/rocket-factory/iam/internal/repository/converter"
	"github.com/vixart/rocket-factory/iam/internal/repository/redis_view"
)

func (r *repository) Get(ctx context.Context, sessionUuid uuid.UUID) (model.User, model.Session, error) {
	sessionKey := r.getSessionKey(sessionUuid.String())

	var sessionView redis_view.SessionRedisView
	err := r.client.HGetAll(ctx, sessionKey).Scan(&sessionView)
	if err != nil {
		// HGetAll never returns redis.Nil: only string commands (GET, HGET, ...)
		// report a missing key that way. This branch only guards against someone
		// passing Nil down from above (a client wrapper, for example). The real
		// "not found" case of HGetAll is caught by the empty UUID check below.
		//
		if errors.Is(err, redis.Nil) {
			return model.User{}, model.Session{}, errs.ErrSessionNotFound
		}

		return model.User{}, model.Session{}, err
	}

	// For a missing key HGetAll returns an empty map and NO error. Checking the
	// primary key field of our view is the only way to tell a missing key from a
	// key that exists but is empty.
	if sessionView.SessionUUID == "" {
		return model.User{}, model.Session{}, errs.ErrSessionNotFound
	}

	user, session := converter.SessionViewToModels(sessionView)

	return user, session, nil
}
