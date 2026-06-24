package converter

import (
	"time"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/iam/internal/model"
	"github.com/vixart/rocket-factory/iam/internal/repository/redis_view"
)

func UserModelToSessionView(sessionUUID uuid.UUID, user model.User, expiredAt time.Time) redis_view.SessionRedisView {
	view := redis_view.SessionRedisView{
		UserUUID:           user.UUID.String(),
		UserLogin:          user.Login,
		UserPasswordHash:   user.PasswordHash,
		UserCreatedAtNS:    user.CreatedAt.UnixNano(),
		SessionUUID:        sessionUUID.String(),
		SessionCreatedAtNS: time.Now().UnixNano(),
		SessionExpiresAtNS: expiredAt.UnixNano(),
	}

	if user.UpdatedAt != nil {
		view.UserUpdatedAtNS = new(user.UpdatedAt.UnixNano())
	}

	return view
}

func SessionViewToUserModel(sessionView redis_view.SessionRedisView) model.User {
	user := model.User{
		UUID:         uuid.MustParse(sessionView.UserUUID),
		Login:        sessionView.UserLogin,
		PasswordHash: sessionView.UserPasswordHash,
		CreatedAt:    time.Unix(0, sessionView.UserCreatedAtNS),
	}

	if sessionView.UserUpdatedAtNS != nil {
		user.UpdatedAt = new(time.Unix(0, sessionView.UserCreatedAtNS))
	}

	return user
}

func SessionViewToSessionModel(sessionView redis_view.SessionRedisView) model.Session {
	session := model.Session{
		UUID:      uuid.MustParse(sessionView.SessionUUID),
		UserUUID:  uuid.MustParse(sessionView.UserUUID),
		Login:     sessionView.UserLogin,
		CreatedAt: time.Unix(0, sessionView.SessionCreatedAtNS),
		ExpiresAt: time.Unix(0, sessionView.SessionExpiresAtNS),
	}

	if sessionView.SessionUpdatedAtNS != nil {
		session.UpdatedAt = new(time.Unix(0, *sessionView.SessionUpdatedAtNS))
	}

	return session
}

func SessionViewToModels(sessionView redis_view.SessionRedisView) (model.User, model.Session) {
	user := SessionViewToUserModel(sessionView)
	session := SessionViewToSessionModel(sessionView)

	return user, session
}
