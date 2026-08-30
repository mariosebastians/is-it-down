package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"isitdown/internal/config"
	"isitdown/internal/db"
	"isitdown/internal/scheduler"
	"isitdown/internal/store"
)

func main() {
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	s := store.New(pool)
	sched := scheduler.New(s, 10*time.Second)

	log.Println("worker started, polling monitors")
	sched.Run(ctx)
	log.Println("worker shutting down")
}
