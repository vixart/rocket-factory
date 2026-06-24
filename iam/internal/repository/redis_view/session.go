package redis_view

type SessionRedisView struct {
	UserUUID           string `redis:"user_uuid"`
	UserLogin          string `redis:"user_login"`
	UserPasswordHash   string `redis:"user_password_hash"`
	UserCreatedAtNS    int64  `redis:"user_created_at"`
	UserUpdatedAtNS    *int64 `redis:"user_updated_at,omitempty"`
	SessionUUID        string `redis:"session_uuid"`
	SessionCreatedAtNS int64  `redis:"session_created_at"`
	SessionUpdatedAtNS *int64 `redis:"session_updated_at,omitempty"`
	SessionExpiresAtNS int64  `redis:"session_expired_at"`
}
