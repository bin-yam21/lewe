package router

import (
	"net/http"

	"github.com/yeabt/lewe/internal/items"
	"github.com/yeabt/lewe/internal/matching"
	"github.com/yeabt/lewe/internal/middleware"
	"github.com/yeabt/lewe/internal/ratings"
	"github.com/yeabt/lewe/internal/uploads"
	"github.com/yeabt/lewe/internal/users"
)

// New creates and configures the HTTP router with all application routes.
func New(userHandler *users.Handler, itemHandler *items.Handler, matchHandler *matching.Handler, ratingHandler *ratings.Handler, uploadHandler *uploads.Handler, uploadDir string, jwtSecret string) http.Handler {
	mux := http.NewServeMux()

	authMW := middleware.Auth(jwtSecret)

	// --- Public auth routes ---
	mux.HandleFunc("POST /api/v1/auth/register", userHandler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", userHandler.Login)
	mux.HandleFunc("POST /api/v1/auth/refresh", userHandler.RefreshToken)

	// --- Protected user routes ---
	mux.Handle("GET /api/v1/users/me", authMW(http.HandlerFunc(userHandler.GetProfile)))
	mux.Handle("PUT /api/v1/users/me", authMW(http.HandlerFunc(userHandler.UpdateProfile)))

	// --- Public item routes ---
	mux.HandleFunc("GET /api/v1/items", itemHandler.List)
	mux.HandleFunc("GET /api/v1/items/{id}", itemHandler.Get)
	mux.HandleFunc("GET /api/v1/items/{id}/similar", itemHandler.Similar)

	// --- Protected item routes ---
	mux.Handle("POST /api/v1/items", authMW(http.HandlerFunc(itemHandler.Create)))
	mux.Handle("PUT /api/v1/items/{id}", authMW(http.HandlerFunc(itemHandler.Update)))
	mux.Handle("DELETE /api/v1/items/{id}", authMW(http.HandlerFunc(itemHandler.Delete)))
	mux.Handle("GET /api/v1/users/me/items", authMW(http.HandlerFunc(itemHandler.ListMine)))

	// --- Protected matching routes ---
	mux.Handle("POST /api/v1/items/{id}/matches", authMW(http.HandlerFunc(matchHandler.FindMatches)))
	mux.Handle("POST /api/v1/items/{id}/offers", authMW(http.HandlerFunc(matchHandler.Offer)))
	mux.Handle("GET /api/v1/matches", authMW(http.HandlerFunc(matchHandler.ListMine)))
	mux.Handle("GET /api/v1/matches/{id}", authMW(http.HandlerFunc(matchHandler.Get)))
	mux.Handle("POST /api/v1/matches/{id}/respond", authMW(http.HandlerFunc(matchHandler.Respond)))
	mux.Handle("POST /api/v1/matches/{id}/complete", authMW(http.HandlerFunc(matchHandler.Complete)))
	mux.Handle("POST /api/v1/matches/{id}/cancel", authMW(http.HandlerFunc(matchHandler.Cancel)))

	// --- Rating routes ---
	mux.Handle("POST /api/v1/matches/{id}/rate", authMW(http.HandlerFunc(ratingHandler.Rate)))
	mux.Handle("GET /api/v1/matches/{id}/ratings", authMW(http.HandlerFunc(ratingHandler.GetMatchRatings)))
	mux.HandleFunc("GET /api/v1/users/{id}/rating", ratingHandler.GetUserRating)

	// --- Uploads ---
	mux.Handle("POST /api/v1/uploads", authMW(http.HandlerFunc(uploadHandler.Upload)))

	// Stored images are served straight off disk. Public by design: an item
	// photo is shown on the public feed, so it carries no more access control
	// than the listing itself. Filenames are 128 bits of randomness, so they
	// are not guessable.
	mux.Handle("GET /uploads/", http.StripPrefix("/uploads/",
		http.FileServer(http.Dir(uploadDir)),
	))

	// --- Health check ---
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Wrap entire mux with logging
	return middleware.Logging(mux)
}
