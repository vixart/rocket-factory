package session

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisClient interface {
	HSet(ctx context.Context, key string, values ...any) *redis.IntCmd
	HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd
	ExpireAt(ctx context.Context, key string, tm time.Time) *redis.BoolCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}
