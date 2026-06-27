package model

import (
	"time"

	"github.com/google/uuid"
)

// User — доменная модель пользователя.
type User struct {
	UUID         uuid.UUID
	Login        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    *time.Time
}
