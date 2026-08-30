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

func main() {
	cfg := config.Load()

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	s := store.New(pool)
	srv := httpapi.NewServer(s)

	log.Printf("api listening on %s", cfg.APIAddr)
	if err := http.ListenAndServe(cfg.APIAddr, srv); err != nil {
		log.Fatalf("api server: %v", err)
	}
}
