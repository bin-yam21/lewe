package router

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/yeabt/lewe/internal/items"
	"github.com/yeabt/lewe/internal/matches"
	"github.com/yeabt/lewe/internal/messages"
	"github.com/yeabt/lewe/internal/middleware"
	"github.com/yeabt/lewe/internal/notifications"
	"github.com/yeabt/lewe/internal/ratings"
	"github.com/yeabt/lewe/internal/response"
	"github.com/yeabt/lewe/internal/uploads"
	"github.com/yeabt/lewe/internal/users"
	"github.com/yeabt/lewe/internal/wants"
)

// Handlers groups the domain handlers the router dispatches to.
type Handlers struct {
	Users         *users.Handler
	Items         *items.Handler
	Wants         *wants.Handler
	Matches       *matches.Handler
	Ratings       *ratings.Handler
	Messages      *messages.Handler
	Notifications *notifications.Handler
	Uploads       *uploads.Handler
}

// Options configures cross-cutting HTTP behaviour.
type Options struct {
	JWTSecret     string
	AuthRateLimit int // requests per IP per minute on /auth routes; 0 disables
	TrustProxy    bool
	CORSOrigins   []string
	// WebAppDir, if it holds a built Mini App (index.html), is served at "/".
	WebAppDir string
}

// New creates and configures the HTTP router with all application routes.
func New(h Handlers, opts Options) http.Handler {
	mux := http.NewServeMux()

	authed := func(fn http.HandlerFunc) http.Handler { return middleware.Auth(opts.JWTSecret)(fn) }
	optional := func(fn http.HandlerFunc) http.Handler { return middleware.OptionalAuth(opts.JWTSecret)(fn) }

	// Auth routes are rate limited per client IP to slow down credential stuffing.
	limited := func(fn http.HandlerFunc) http.Handler { return fn }
	if opts.AuthRateLimit > 0 {
		limiter := middleware.NewRateLimiter(opts.AuthRateLimit, opts.TrustProxy)
		limited = func(fn http.HandlerFunc) http.Handler { return limiter.Middleware(fn) }
	}

	// --- Public auth routes ---
	mux.Handle("POST /api/v1/auth/register", limited(h.Users.Register))
	mux.Handle("POST /api/v1/auth/login", limited(h.Users.Login))
	mux.Handle("POST /api/v1/auth/refresh", limited(h.Users.RefreshToken))
	mux.Handle("POST /api/v1/auth/logout", limited(h.Users.Logout))
	mux.Handle("POST /api/v1/auth/telegram", limited(h.Users.TelegramLogin))
	mux.Handle("POST /api/v1/auth/verify-email", limited(h.Users.VerifyEmail))
	mux.Handle("POST /api/v1/auth/forgot-password", limited(h.Users.ForgotPassword))
	mux.Handle("POST /api/v1/auth/reset-password", limited(h.Users.ResetPassword))

	// --- Users ---
	mux.Handle("GET /api/v1/users/me", authed(h.Users.GetProfile))
	mux.Handle("PUT /api/v1/users/me", authed(h.Users.UpdateProfile))
	mux.Handle("PUT /api/v1/users/me/password", authed(h.Users.ChangePassword))
	mux.Handle("POST /api/v1/users/me/verify-email", authed(h.Users.ResendVerification))
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
	mux.Handle("GET /api/v1/matches/{id}/messages", authed(h.Messages.List))
	mux.Handle("POST /api/v1/matches/{id}/messages", authed(h.Messages.Send))

	// --- Notifications ---
	mux.Handle("GET /api/v1/notifications", authed(h.Notifications.List))
	mux.Handle("GET /api/v1/notifications/unread-count", authed(h.Notifications.UnreadCount))
	mux.Handle("POST /api/v1/notifications/read-all", authed(h.Notifications.MarkAllRead))
	mux.Handle("POST /api/v1/notifications/{id}/read", authed(h.Notifications.MarkRead))

	// --- Uploads ---
	mux.Handle("POST /api/v1/uploads", authed(h.Uploads.Upload))
	mux.HandleFunc("GET /uploads/{name}", h.Uploads.Serve)

	// --- Telegram Mini App ---
	if opts.WebAppDir != "" {
		if _, err := os.Stat(filepath.Join(opts.WebAppDir, "index.html")); err == nil {
			mux.Handle("GET /", spa(opts.WebAppDir))
		}
	}

	// --- Health check ---
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Wrap entire mux with logging, CORS and panic recovery
	var handler http.Handler = middleware.Recover(mux)
	if len(opts.CORSOrigins) > 0 {
		handler = middleware.CORS(opts.CORSOrigins)(handler)
	}
	return middleware.Logging(handler)
}
