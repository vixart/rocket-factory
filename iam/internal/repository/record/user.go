package record

import (
	"time"

	"github.com/google/uuid"
)

// User — доменная модель пользователя.
type User struct {
	UUID         uuid.UUID  `db:"uuid"`
	Login        string     `db:"login"`
	PasswordHash string     `db:"password_hash"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    *time.Time `db:"updated_at"`
}
