package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"go-react-shadcn/internal/config"
	"go-react-shadcn/internal/httpserver"
	"go-react-shadcn/internal/migrate"
	"go-react-shadcn/internal/store"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		slog.Error("invalid config", "error", err)
		os.Exit(1)
	}
	if err := migrate.Up(cfg.DatabasePath); err != nil {
		slog.Error("migrate failed", "error", err)
		os.Exit(1)
	}
	db, err := store.Open(cfg.DatabasePath)
	if err != nil {
		slog.Error("database failed", "error", err)
		os.Exit(1)
	}
	app, err := httpserver.New(cfg, db)
	if err != nil {
		slog.Error("server init failed", "error", err)
		os.Exit(1)
	}
	addr := ":" + cfg.Port
	slog.Info("latch api listening", "addr", addr, "db", cfg.DatabasePath)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := app.ListenAndServe(ctx, addr); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
