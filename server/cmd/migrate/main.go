package main

import (
	"log/slog"
	"os"

	"go-react-shadcn/internal/config"
	"go-react-shadcn/internal/migrate"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	cfg := config.Load()
	if err := migrate.Up(cfg.DatabasePath); err != nil {
		slog.Error("migrate failed", "error", err)
		os.Exit(1)
	}
	slog.Info("migrate ok", "db", cfg.DatabasePath)
}
