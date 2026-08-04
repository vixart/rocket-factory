package order

import (
	"context"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TxManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type repository struct {
	pool      *pgxpool.Pool
	getter    *trmpgx.CtxGetter
	txManager TxManager
}

func New(pool *pgxpool.Pool, txManager TxManager) *repository {
	return &repository{
		pool:      pool,
		getter:    trmpgx.DefaultCtxGetter,
		txManager: txManager,
	}
}
