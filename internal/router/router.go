package router

import (
	"net/http"

	"github.com/yeabt/lewe/internal/middleware"
	"github.com/yeabt/lewe/internal/users"
)

// New creates and configures the HTTP router with all application routes.
func New(userHandler *users.Handler, jwtSecret string) http.Handler {
	mux := http.NewServeMux()

	authMW := middleware.Auth(jwtSecret)

	// --- Public auth routes ---
	mux.HandleFunc("POST /api/v1/auth/register", userHandler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", userHandler.Login)
	mux.HandleFunc("POST /api/v1/auth/refresh", userHandler.RefreshToken)

	// --- Protected user routes ---
	mux.Handle("GET /api/v1/users/me", authMW(http.HandlerFunc(userHandler.GetProfile)))
	mux.Handle("PUT /api/v1/users/me", authMW(http.HandlerFunc(userHandler.UpdateProfile)))

	// --- Health check ---
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Wrap entire mux with logging
	return middleware.Logging(mux)
}
