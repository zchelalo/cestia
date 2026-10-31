package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zchelalo/cestia/apps/api/internal/api"
	"github.com/zchelalo/cestia/apps/api/internal/config"
	"github.com/zchelalo/cestia/apps/api/internal/data"
	"github.com/zchelalo/cestia/apps/api/internal/mailer"
	"github.com/zchelalo/cestia/apps/api/internal/modules/groups"
	"github.com/zchelalo/cestia/apps/api/internal/modules/identity"
	"github.com/zchelalo/cestia/apps/api/internal/modules/notifications"
	"github.com/zchelalo/cestia/apps/api/internal/modules/shopping"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid configuration", "stage", "load_config", "error", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	store, err := data.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database unavailable", "stage", "connect_database", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	identityService := identity.New(store, cfg.TokenSecret, cfg.SessionTTL, cfg.RegisterRateLimit)
	groupService := groups.New(store, cfg.TokenSecret)
	shoppingService := shopping.New(store)
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.New(cfg, identityService, groupService, shoppingService, logger).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	sender := mailer.New(mailer.SMTP{
		Host: cfg.SMTP.Host, Port: cfg.SMTP.Port, Username: cfg.SMTP.Username,
		Password: cfg.SMTP.Password, From: cfg.SMTP.From, TLS: cfg.SMTP.TLS, StartTLS: cfg.SMTP.StartTLS,
	})
	worker := notifications.NewWorker(store, sender, cfg.TokenSecret, cfg.AppOrigin, logger)
	go worker.Run(ctx, 2*time.Second)

	go func() {
		logger.Info("api listening", "addr", cfg.HTTPAddr, "env", cfg.Env)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "stage", "serve", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "stage", "shutdown", "error", err)
	}
}
