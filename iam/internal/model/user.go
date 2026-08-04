package model

import (
	"time"

	"github.com/google/uuid"
)

// User is the domain model of a user.
type User struct {
	UUID         uuid.UUID
	Login        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    *time.Time
}
