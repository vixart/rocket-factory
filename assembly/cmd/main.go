package main

import (
	"context"
	"log/slog"

	"github.com/joho/godotenv"

	"github.com/vixart/rocket-factory/assembly/internal/app"
	"github.com/vixart/rocket-factory/assembly/internal/config"
)

func main() {
	_ = godotenv.Load("assembly.env") //nolint:gosec // the .env file is optional, a load error is fine

	config.MustLoad()

	a := app.New(context.Background())

	if err := a.Run(); err != nil {
		slog.Error("application failed", "error", err)
	}
}
