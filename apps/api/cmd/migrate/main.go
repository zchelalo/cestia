package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/zchelalo/cestia/apps/api/internal/config"
	"github.com/zchelalo/cestia/apps/api/internal/data"
	"github.com/zchelalo/cestia/apps/api/internal/migrations"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	ctx := context.Background()
	store, err := data.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database unavailable", "error", err)
		os.Exit(1)
	}
	defer store.Close()
	if err := migrations.Up(ctx, store.Pool); err != nil {
		logger.Error("migration failed", "error", err)
		os.Exit(1)
	}
	logger.Info("migrations applied")
}
