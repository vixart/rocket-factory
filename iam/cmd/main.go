package main

import (
	"context"
	"log/slog"

	"github.com/joho/godotenv"

	"github.com/vixart/rocket-factory/iam/internal/app"
	"github.com/vixart/rocket-factory/iam/internal/config"
)

func main() {
	// Load environment variables from iam.env when the file exists
	_ = godotenv.Load("iam.env") //nolint:gosec // the .env file is optional, a load error is fine

	config.MustLoad()

	a := app.New(context.Background())

	if err := a.Run(); err != nil {
		slog.Error("application failed", "error", err)
	}
}
