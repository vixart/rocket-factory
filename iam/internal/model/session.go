package model

import (
	"time"

	"github.com/google/uuid"
)

// Session is the domain model of a session.
// It is stored in Redis as a hash with a TTL.
type Session struct {
	UUID      uuid.UUID
	UserUUID  uuid.UUID
	Login     string
	CreatedAt time.Time
	UpdatedAt *time.Time
	ExpiresAt time.Time
}
