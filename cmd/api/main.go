package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yeabt/lewe/internal/config"
	"github.com/yeabt/lewe/internal/db"
	"github.com/yeabt/lewe/internal/items"
	"github.com/yeabt/lewe/internal/matching"
	"github.com/yeabt/lewe/internal/ratings"
	"github.com/yeabt/lewe/internal/router"
	"github.com/yeabt/lewe/internal/users"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Run database migrations
	db.RunMigrations(cfg.DatabaseURL, "file://internal/db/migrations")

	// Connect to database
	pool := db.Connect(cfg.DatabaseURL)
	defer pool.Close()

	// Wire up dependencies: repo → service → handler

	// Users domain
	userRepo := users.NewRepository(pool)
	tokenRepo := users.NewRefreshTokenRepository(pool)
	userSvc := users.NewService(userRepo, tokenRepo, cfg.JWTSecret)
	userHandler := users.NewHandler(userSvc)

	// Items domain
	itemRepo := items.NewRepository(pool)
	itemSvc := items.NewService(itemRepo)
	itemHandler := items.NewHandler(itemSvc)

	// Matching domain
	matchRepo := matching.NewRepository(pool)
	matchSvc := matching.NewService(matchRepo, itemRepo)
	matchHandler := matching.NewHandler(matchSvc)

	// Ratings domain
	ratingRepo := ratings.NewRepository(pool)
	ratingSvc := ratings.NewService(ratingRepo, matchSvc)
	ratingHandler := ratings.NewHandler(ratingSvc)

	// Build router
	r := router.New(userHandler, itemHandler, matchHandler, ratingHandler, cfg.JWTSecret)

	// Configure HTTP server
	srv := &http.Server{
		Addr:         cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Lewe API server starting on %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Graceful shutdown on SIGINT / SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
