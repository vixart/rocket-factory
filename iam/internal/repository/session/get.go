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
		// redis.Nil от HGetAll не приходит — эту ошибку возвращают только
		// строковые команды (GET, HGET и т.п.) для отсутствующего ключа.
		// Здесь ветка нужна только на случай, если кто-то прокинет Nil
		// сверху (например, обёрткой клиента). Реальный «not found»
		// у HGetAll ловится проверкой на пустой UUID ниже.
		if errors.Is(err, redis.Nil) {
			return model.User{}, model.Session{}, errs.ErrSessionNotFound
		}

		return model.User{}, model.Session{}, err
	}

	// HGetAll для несуществующего ключа возвращает пустую map БЕЗ ошибки.
	// Это единственный способ отличить «нет такого ключа» от «ключ есть, но пустой» —
	// проверяем обязательное поле первичного ключа в нашей view.
	if sessionView.SessionUUID == "" {
		return model.User{}, model.Session{}, errs.ErrSessionNotFound
	}

	user, session := converter.SessionViewToModels(sessionView)

	return user, session, nil
}
