package model

import (
	"time"

	"github.com/google/uuid"
)

// Session — доменная модель сессии.
// Хранится в Redis как HashMap с TTL.
type Session struct {
	UUID      uuid.UUID
	UserUUID  uuid.UUID
	Login     string
	CreatedAt time.Time
	UpdatedAt *time.Time
	ExpiresAt time.Time
}
