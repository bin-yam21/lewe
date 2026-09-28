package router

import (
	"net/http"

	"github.com/yeabt/lewe/internal/items"
	"github.com/yeabt/lewe/internal/matches"
	"github.com/yeabt/lewe/internal/middleware"
	"github.com/yeabt/lewe/internal/ratings"
	"github.com/yeabt/lewe/internal/response"
	"github.com/yeabt/lewe/internal/users"
	"github.com/yeabt/lewe/internal/wants"
)

// Handlers groups the domain handlers the router dispatches to.
type Handlers struct {
	Users   *users.Handler
	Items   *items.Handler
	Wants   *wants.Handler
	Matches *matches.Handler
	Ratings *ratings.Handler
}

// New creates and configures the HTTP router with all application routes.
func New(h Handlers, jwtSecret string) http.Handler {
	mux := http.NewServeMux()

	authed := func(fn http.HandlerFunc) http.Handler { return middleware.Auth(jwtSecret)(fn) }
	optional := func(fn http.HandlerFunc) http.Handler { return middleware.OptionalAuth(jwtSecret)(fn) }

	// --- Public auth routes ---
	mux.HandleFunc("POST /api/v1/auth/register", h.Users.Register)
	mux.HandleFunc("POST /api/v1/auth/login", h.Users.Login)
	mux.HandleFunc("POST /api/v1/auth/refresh", h.Users.RefreshToken)
	mux.HandleFunc("POST /api/v1/auth/logout", h.Users.Logout)

	// --- Users ---
	mux.Handle("GET /api/v1/users/me", authed(h.Users.GetProfile))
	mux.Handle("PUT /api/v1/users/me", authed(h.Users.UpdateProfile))
	mux.Handle("GET /api/v1/users/me/items", authed(h.Items.ListMine))
	mux.HandleFunc("GET /api/v1/users/{id}", h.Users.GetPublicProfile)
	mux.HandleFunc("GET /api/v1/users/{id}/ratings", h.Ratings.ListForUser)

	// --- Catalog ---
	mux.HandleFunc("GET /api/v1/categories", h.Items.Categories)

	// --- Items ---
	mux.HandleFunc("GET /api/v1/items", h.Items.List)
	mux.Handle("POST /api/v1/items", authed(h.Items.Create))
	mux.Handle("GET /api/v1/items/{id}", optional(h.Items.Get))
	mux.Handle("PUT /api/v1/items/{id}", authed(h.Items.Update))
	mux.Handle("DELETE /api/v1/items/{id}", authed(h.Items.Delete))

	// --- Wants ---
	mux.Handle("GET /api/v1/wants", authed(h.Wants.List))
	mux.Handle("POST /api/v1/wants", authed(h.Wants.Create))
	mux.Handle("GET /api/v1/wants/{id}", authed(h.Wants.Get))
	mux.Handle("PUT /api/v1/wants/{id}", authed(h.Wants.Update))
	mux.Handle("DELETE /api/v1/wants/{id}", authed(h.Wants.Delete))

	// --- Matches ---
	mux.Handle("GET /api/v1/matches", authed(h.Matches.List))
	mux.Handle("GET /api/v1/matches/{id}", authed(h.Matches.Get))
	mux.Handle("POST /api/v1/matches/{id}/accept", authed(h.Matches.Accept))
	mux.Handle("POST /api/v1/matches/{id}/decline", authed(h.Matches.Decline))
	mux.Handle("PUT /api/v1/matches/{id}/exchange", authed(h.Matches.SetExchange))
	mux.Handle("POST /api/v1/matches/{id}/complete", authed(h.Matches.Complete))
	mux.Handle("POST /api/v1/matches/{id}/cancel", authed(h.Matches.Cancel))
	mux.Handle("POST /api/v1/matches/{id}/rating", authed(h.Ratings.Rate))

	// --- Health check ---
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Wrap entire mux with panic recovery and logging
	return middleware.Logging(middleware.Recover(mux))
}
