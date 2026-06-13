package main

import (
	"context"
	"log/slog"

	"github.com/joho/godotenv"

	"github.com/vixart/rocket-factory/assembly/internal/app"
	"github.com/vixart/rocket-factory/assembly/internal/config"
)

func main() {
	_ = godotenv.Load("assembly.env") //nolint:gosec // .env файл опционален — ошибка загрузки допустима

	config.MustLoad()

	a := app.New(context.Background())

	if err := a.Run(); err != nil {
		slog.Error("ошибка при работе приложения", "error", err)
	}
}
