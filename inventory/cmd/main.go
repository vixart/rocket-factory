package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	inventoryApiV1 "github.com/vixart/rocket-factory/inventory/internal/api/inventory/v1"
	"github.com/vixart/rocket-factory/inventory/internal/model"
	partRepository "github.com/vixart/rocket-factory/inventory/internal/repository/part"
	"github.com/vixart/rocket-factory/inventory/internal/repository/record"
	"github.com/vixart/rocket-factory/inventory/internal/service/part"
	inventoryv1 "github.com/vixart/rocket-factory/shared/pkg/proto/inventory/v1"
)

const (
	// Адрес сервера.
	grpcAddress = "localhost:50051"

	// gRPC keepalive параметры.
	grpcMaxConnectionIdle     = 15 * time.Minute // Закрыть idle-соединения (нет активных RPC)
	grpcMaxConnectionAge      = 30 * time.Minute // Принудительная ротация для балансировки
	grpcMaxConnectionAgeGrace = 5 * time.Second  // Время на завершение активных RPC
	grpcKeepaliveTime         = 5 * time.Minute  // Интервал ping'ов для обнаружения мёртвых соединений
	grpcKeepaliveTimeout      = 1 * time.Second  // Тайм-аут ожидания pong
	grpcMinPingInterval       = 5 * time.Minute  // Минимальный интервал ping'ов от клиента (защита от DoS)
)

func main() {
	ctx := context.Background()
	lc := net.ListenConfig{}
	lis, err := lc.Listen(ctx, "tcp", grpcAddress)
	if err != nil {
		slog.Error("не удалось создать listener", "error", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     grpcMaxConnectionIdle,
			MaxConnectionAge:      grpcMaxConnectionAge,
			MaxConnectionAgeGrace: grpcMaxConnectionAgeGrace,
			Time:                  grpcKeepaliveTime,
			Timeout:               grpcKeepaliveTimeout,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             grpcMinPingInterval,
			PermitWithoutStream: true, // Разрешить "тёплые" соединения без активных RPC
		}),
	)

	repo := partRepository.NewRepository(new(composeSeedData()))
	service := part.NewService(repo)
	api := inventoryApiV1.NewApi(service)

	inventoryv1.RegisterInventoryServiceServer(grpcServer, api)

	// Включаем reflection для postman/grpcurl
	reflection.Register(grpcServer)

	slog.Info("запуск InventoryService", "адрес", grpcAddress)

	go func() {
		slog.Info("🚀 gRPC сервер запущен", "address", grpcAddress)
		if serveErr := grpcServer.Serve(lis); serveErr != nil {
			slog.Error("ошибка запуска сервера", "error", serveErr)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("🛑 остановка gRPC сервера")
	grpcServer.GracefulStop()
	slog.Info("✅ сервер остановлен")
}

func composeSeedData() map[uuid.UUID]record.Part {
	now := new(time.Now())

	return map[uuid.UUID]record.Part{
		uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"): {
			UUID:          uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
			Name:          "Алюминиевый корпус",
			Description:   "Лёгкий корпус для небольших кораблей",
			Price:         500000,
			PartType:      model.PartTypeHull,
			StockQuantity: 10,
			CreatedAt:     now,
		},
		uuid.MustParse("550e8400-e29b-41d4-a716-446655440002"): {
			UUID:          uuid.MustParse("550e8400-e29b-41d4-a716-446655440002"),
			Name:          "Титановый корпус",
			Description:   "Прочный корпус для средних кораблей",
			Price:         1500000,
			PartType:      model.PartTypeHull,
			StockQuantity: 5,
			CreatedAt:     now,
		},
		uuid.MustParse("550e8400-e29b-41d4-a716-446655440003"): {
			UUID:          uuid.MustParse("550e8400-e29b-41d4-a716-446655440003"),
			Name:          "Ионный двигатель C",
			Description:   "Базовый ионный двигатель класса C",
			Price:         300000,
			PartType:      model.PartTypeEngine,
			StockQuantity: 8,
			CreatedAt:     now,
		},
		uuid.MustParse("550e8400-e29b-41d4-a716-446655440004"): {
			UUID:          uuid.MustParse("550e8400-e29b-41d4-a716-446655440004"),
			Name:          "Ионный двигатель B",
			Description:   "Улучшенный ионный двигатель класса B",
			Price:         800000,
			PartType:      model.PartTypeEngine,
			StockQuantity: 3,
			CreatedAt:     now,
		},
		uuid.MustParse("550e8400-e29b-41d4-a716-446655440005"): {
			UUID:          uuid.MustParse("550e8400-e29b-41d4-a716-446655440005"),
			Name:          "Энергетический щит",
			Description:   "Стандартный энергетический щит",
			Price:         400000,
			PartType:      model.PartTypeShield,
			StockQuantity: 6,
			CreatedAt:     now,
		},
		uuid.MustParse("550e8400-e29b-41d4-a716-446655440006"): {
			UUID:          uuid.MustParse("550e8400-e29b-41d4-a716-446655440006"),
			Name:          "Лазерная пушка",
			Description:   "Точная лазерная пушка",
			Price:         250000,
			PartType:      model.PartTypeWeapon,
			StockQuantity: 7,
			CreatedAt:     now,
		},
		uuid.MustParse("550e8400-e29b-41d4-a716-446655440007"): {
			UUID:          uuid.MustParse("550e8400-e29b-41d4-a716-446655440007"),
			Name:          "Плазменный корпус",
			Description:   "Экспериментальный корпус (нет на складе)",
			Price:         2000000,
			PartType:      model.PartTypeHull,
			StockQuantity: 0,
			CreatedAt:     now,
		},
	}
}
