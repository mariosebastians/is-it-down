package main

import (
	"context"
	"log"
	"net/http"

	"isitdown/internal/config"
	"isitdown/internal/db"
	"isitdown/internal/httpapi"
	"isitdown/internal/store"
)

// @title           is-it-down API
// @version         1.0
// @description     Self-hosted uptime and status monitoring API. Manages monitors and exposes their check history and live status.
// @license.name    MIT
// @license.url     https://opensource.org/licenses/MIT
// @host            localhost:8080
// @BasePath        /
func main() {
	cfg := config.Load()

	ctx := context.Background()
	gdb, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	if err := db.AutoMigrate(gdb, &store.Monitor{}, &store.Check{}); err != nil {
		log.Fatalf("migrate database: %v", err)
	}

	s := store.New(gdb)
	srv := httpapi.NewServer(s)

	log.Printf("api listening on %s", cfg.APIAddr)
	if err := http.ListenAndServe(cfg.APIAddr, srv); err != nil {
		log.Fatalf("api server: %v", err)
	}
}
