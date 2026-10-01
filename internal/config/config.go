package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/yeabt/lewe/internal/mail"
)

// Config holds all configuration for the application.
type Config struct {
	DatabaseURL   string
	JWTSecret     string
	Port          string
	MatchInterval time.Duration
	// AuthRateLimit is the number of auth requests allowed per client IP per
	// minute; 0 disables rate limiting.
	AuthRateLimit int
	// TrustProxy takes the client IP from X-Forwarded-For (only behind a proxy).
	TrustProxy bool
	// CORSOrigins lists browser origins allowed to call the API ("*" for any).
	CORSOrigins []string
	// AppURL is the client app's base URL, used for links in emails.
	AppURL string
	// SMTP is used to send email when SMTP.Host is set; otherwise emails are logged.
	SMTP mail.SMTPConfig
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
		AuthRateLimit: 20,
		TrustProxy:    os.Getenv("TRUST_PROXY") == "true",
		AppURL:        getEnv("APP_URL", "http://localhost:3000"),
		SMTP: mail.SMTPConfig{
			Host:     os.Getenv("SMTP_HOST"),
			Port:     587,
			Username: os.Getenv("SMTP_USERNAME"),
			Password: os.Getenv("SMTP_PASSWORD"),
			From:     getEnv("SMTP_FROM", "Lewe <no-reply@localhost>"),
		},
	}
	if v := os.Getenv("SMTP_PORT"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			log.Fatalf("SMTP_PORT must be a port number, got %q", v)
		}
		cfg.SMTP.Port = n
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

	if v := os.Getenv("AUTH_RATE_LIMIT"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			log.Fatalf("AUTH_RATE_LIMIT must be a non-negative integer, got %q", v)
		}
		cfg.AuthRateLimit = n
	}
	for _, o := range strings.Split(os.Getenv("CORS_ALLOWED_ORIGINS"), ",") {
		if o = strings.TrimSpace(o); o != "" {
			cfg.CORSOrigins = append(cfg.CORSOrigins, o)
		}
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
