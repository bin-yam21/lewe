package config

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application.
type Config struct {
	DatabaseURL   string
	JWTSecret     string
	Port          string
	MatchInterval time.Duration
}

// Load reads configuration from environment variables.
// It attempts to load a .env file first, but does not fail if one is absent
// (production environments typically inject env vars directly).
func Load() Config {
	// Best-effort .env load — ignore error (file may not exist in prod)
	_ = godotenv.Load()

	cfg := Config{
		DatabaseURL:   getEnv("DATABASE_URL", ""),
		JWTSecret:     getEnv("JWT_SECRET", ""),
		Port:          getEnv("PORT", ":8080"),
		MatchInterval: time.Minute,
	}

	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}
	if len(cfg.JWTSecret) < 32 {
		log.Println("WARNING: JWT_SECRET is shorter than 32 characters; use a longer random secret in production")
	}
	// Accept both "8080" and ":8080".
	if cfg.Port[0] != ':' {
		cfg.Port = ":" + cfg.Port
	}
	if v := os.Getenv("MATCH_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 {
			log.Fatalf("MATCH_INTERVAL must be a positive duration such as 30s or 5m, got %q", v)
		}
		cfg.MatchInterval = d
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
