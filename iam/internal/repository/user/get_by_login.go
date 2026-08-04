package user

import (
	"context"
	"fmt"

	"github.com/go-faster/errors"
	"github.com/jackc/pgx/v5"

	errs "github.com/vixart/rocket-factory/iam/internal/errors"
	"github.com/vixart/rocket-factory/iam/internal/model"
	"github.com/vixart/rocket-factory/iam/internal/repository/converter"
	"github.com/vixart/rocket-factory/iam/internal/repository/record"
)

func (r *repository) GetByLogin(ctx context.Context, login string) (model.User, error) {
	const queryOrder = `SELECT uuid, login, password_hash, created_at, updated_at FROM users WHERE login = $1`

	var user record.User

	err := r.getter.DefaultTrOrDB(ctx, r.pool).QueryRow(ctx, queryOrder, login).Scan(
		&user.UUID, &user.Login, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, errs.ErrUserNotFound
		}
		return model.User{}, fmt.Errorf("fetch user: %w", err)
	}

	return converter.UserRecordToModel(user), nil
}
