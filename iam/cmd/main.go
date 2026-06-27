package main

import (
	"context"
	"log/slog"

	"github.com/joho/godotenv"

	"github.com/vixart/rocket-factory/iam/internal/app"
	"github.com/vixart/rocket-factory/iam/internal/config"
)

func main() {
	// Загружаем переменные окружения из iam.env (если файл существует)
	_ = godotenv.Load("iam.env") //nolint:gosec // .env файл опционален — ошибка загрузки допустима

	config.MustLoad()

	a := app.New(context.Background())

	if err := a.Run(); err != nil {
		slog.Error("ошибка при работе приложения", "error", err)
	}
}
