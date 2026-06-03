package app

import (
	"context"
	"log/slog"
	"os"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"github.com/jackc/pgx/v5/pgxpool"

	inventoryApiV1 "github.com/vixart/rocket-factory/inventory/internal/api/inventory/v1"
	"github.com/vixart/rocket-factory/inventory/internal/config"
	partRepo "github.com/vixart/rocket-factory/inventory/internal/repository/part"
	partService "github.com/vixart/rocket-factory/inventory/internal/service/application/part"
	"github.com/vixart/rocket-factory/inventory/internal/service/domain"
	"github.com/vixart/rocket-factory/platform/pkg/closer"
	inventoryv1 "github.com/vixart/rocket-factory/shared/pkg/proto/inventory/v1"
)

type diContainer struct {
	// Инфраструктура
	pgPool    *pgxpool.Pool
	txManager partRepo.TxManager

	// Зависимости приложения
	partRepo                 partService.Repository
	partCompatibilityChecker partService.CompatibilityChecker

	// Зависимости API
	partSvc inventoryApiV1.InventoryService

	// API-обработчики
	inventoryv1Handler inventoryv1.InventoryServiceServer
}

func (d *diContainer) TxManager(ctx context.Context) partRepo.TxManager {
	if d.txManager == nil {
		txManager, err := manager.New(trmpgx.NewDefaultFactory(d.PGPool(ctx)))
		if err != nil {
			slog.Error("не удалось создать Transaction Manager", "error", err)
			os.Exit(1)
		}
		d.txManager = txManager
	}

	return d.txManager
}

func (d *diContainer) PGPool(ctx context.Context) *pgxpool.Pool {
	if d.pgPool == nil {
		pool, err := pgxpool.New(ctx, config.AppConfig().PG.DSN())
		if err != nil {
			slog.Error("не удалось подключиться к PostgreSQL", "error", err)
			os.Exit(1)
		}

		err = pool.Ping(ctx)
		if err != nil {
			slog.Error("не удалось выполнить ping PostgreSQL", "error", err)
			os.Exit(1)
		}

		closer.Add("PostgreSQL pool", func(_ context.Context) error {
			pool.Close()
			return nil
		})

		d.pgPool = pool
	}

	return d.pgPool
}

func (d *diContainer) PartRepository(ctx context.Context) partService.Repository {
	if d.partRepo == nil {
		d.partRepo = partRepo.NewRepository(d.PGPool(ctx))
	}

	return d.partRepo
}

func (d *diContainer) PartCompatibilityChecker() partService.CompatibilityChecker {
	if d.partCompatibilityChecker == nil {
		d.partCompatibilityChecker = domain.NewCompatibilityChecker()
	}

	return d.partCompatibilityChecker
}

func (d *diContainer) PartService(ctx context.Context) inventoryApiV1.InventoryService {
	if d.partSvc == nil {
		d.partSvc = partService.NewService(d.PartRepository(ctx), d.PartCompatibilityChecker())
	}

	return d.partSvc
}

func (d *diContainer) InventoryV1API(ctx context.Context) inventoryv1.InventoryServiceServer {
	if d.inventoryv1Handler == nil {
		d.inventoryv1Handler = inventoryApiV1.NewApi(d.PartService(ctx))
	}

	return d.inventoryv1Handler
}
