package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/yeabt/lewe/internal/app"
	"github.com/yeabt/lewe/internal/config"
	"github.com/yeabt/lewe/internal/db"
	"github.com/yeabt/lewe/internal/mail"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Run database migrations (embedded in the binary)
	if err := db.RunMigrations(cfg.DatabaseURL); err != nil {
		log.Fatalf("Migrations failed: %v", err)
	}
	log.Println("Migrations applied successfully")

	// Connect to database
	pool := db.Connect(cfg.DatabaseURL)
	defer pool.Close()

	// Email: SMTP when configured, otherwise write messages to the log
	var mailer mail.Mailer = mail.LogMailer{}
	if cfg.SMTP.Host != "" {
		mailer = mail.NewSMTPMailer(cfg.SMTP)
	} else {
		log.Println("SMTP_HOST not set; emails will be written to the log")
	}

	// Wire up dependencies
	a := app.New(pool, app.Config{
		JWTSecret:     cfg.JWTSecret,
		MatchInterval: cfg.MatchInterval,
		AuthRateLimit: cfg.AuthRateLimit,
		TrustProxy:    cfg.TrustProxy,
		CORSOrigins:   cfg.CORSOrigins,
		Mailer:        mailer,
		AppURL:        cfg.AppURL,
	})

	// Cancelled on SIGINT / SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start the background matching worker
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		a.Worker.Run(ctx)
	}()

	// Configure HTTP server
	srv := &http.Server{
		Addr:         cfg.Port,
		Handler:      a.Handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Lewe API server starting on %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Graceful shutdown
	<-ctx.Done()
	log.Println("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}
	wg.Wait()

	log.Println("Server stopped")
}
