package user

import (
	"context"
	"fmt"

	"github.com/go-faster/errors"
	"github.com/jackc/pgx/v5/pgconn"

	errs "github.com/vixart/rocket-factory/iam/internal/errors"
	"github.com/vixart/rocket-factory/iam/internal/model"
	"github.com/vixart/rocket-factory/iam/internal/repository/converter"
)

func (r *repository) Create(ctx context.Context, user model.User) error {
	const query = `INSERT INTO users (uuid, login, password_hash, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`

	userRec := converter.UserModelToRecord(user)

	_, err := r.getter.DefaultTrOrDB(ctx, r.pool).Exec(
		ctx,
		query,
		userRec.UUID,
		userRec.Login,
		userRec.PasswordHash,
		userRec.CreatedAt,
		nil,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return errs.ErrUserAlreadyExists
		}
		return fmt.Errorf("create user: %w", err)
	}

	return nil
}
