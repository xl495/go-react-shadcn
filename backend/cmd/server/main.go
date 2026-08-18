package main

import (
	"log"
	"os"

	"go-react-shadcn/internal/config"
	"go-react-shadcn/internal/httpserver"
	"go-react-shadcn/internal/store"
)

func main() {
	cfg := config.Load()
	db, err := store.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	app, err := httpserver.New(cfg, db)
	if err != nil {
		log.Fatalf("server: %v", err)
	}
	addr := ":" + cfg.Port
	log.Printf("latch api listening on %s (db=%s)", addr, cfg.DatabasePath)
	if err := app.Router.Run(addr); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
