package main

import (
	"context"
	"log/slog"

	"github.com/joho/godotenv"

	"github.com/vixart/rocket-factory/payment/internal/app"
	"github.com/vixart/rocket-factory/payment/internal/config"
)

func main() {
	// Загружаем переменные окружения из ufo.env (если файл существует)
	_ = godotenv.Load("payment.env") //nolint:gosec // .env файл опционален — ошибка загрузки допустима

	config.MustLoad()

	a := app.New(context.Background())

	if err := a.Run(); err != nil {
		slog.Error("ошибка при работе приложения", "error", err)
	}
}
