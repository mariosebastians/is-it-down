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

	gdb, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	if err := db.AutoMigrate(gdb, &store.Monitor{}, &store.Check{}); err != nil {
		log.Fatalf("migrate database: %v", err)
	}

	s := store.New(gdb)
	sched := scheduler.New(s, 10*time.Second)

	log.Println("worker started, polling monitors")
	sched.Run(ctx)
	log.Println("worker shutting down")
}
