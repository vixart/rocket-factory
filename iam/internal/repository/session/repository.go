package session

import "fmt"

const (
	cacheKeyPrefix = "session:"
)

type repository struct {
	client redisClient
}

func NewRepository(client redisClient) *repository {
	return &repository{
		client: client,
	}
}

func (r *repository) getSessionKey(sessionUUID string) string {
	return fmt.Sprintf("%s%s", cacheKeyPrefix, sessionUUID)
}
