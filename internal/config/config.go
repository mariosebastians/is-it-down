package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	APIAddr     string
}

func Load() Config {
	_ = godotenv.Load()

	return Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://isitdown:isitdown@localhost:5432/isitdown?sslmode=disable"),
		APIAddr:     getEnv("API_ADDR", ":8080"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
