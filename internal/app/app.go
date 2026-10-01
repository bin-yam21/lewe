// Package app wires repositories, services, handlers and the router together.
package app

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yeabt/lewe/internal/items"
	"github.com/yeabt/lewe/internal/mail"
	"github.com/yeabt/lewe/internal/matches"
	"github.com/yeabt/lewe/internal/messages"
	"github.com/yeabt/lewe/internal/notifications"
	"github.com/yeabt/lewe/internal/ratings"
	"github.com/yeabt/lewe/internal/router"
	"github.com/yeabt/lewe/internal/uploads"
	"github.com/yeabt/lewe/internal/users"
	"github.com/yeabt/lewe/internal/wants"
)

// App bundles the HTTP handler and the background workers.
type App struct {
	Handler http.Handler
	Worker  *matches.Worker
	// Dispatcher is nil unless a TelegramSender is configured.
	Dispatcher *notifications.Dispatcher
}

// Config holds the settings the application needs beyond a database pool.
type Config struct {
	JWTSecret     string
	MatchInterval time.Duration
	AuthRateLimit int // per client IP per minute; 0 disables
	TrustProxy    bool
	CORSOrigins   []string
	Mailer        mail.Mailer // nil logs emails instead of sending them
	AppURL        string      // base URL of the client app (links in emails and Telegram messages)

	// TelegramBotToken enables Mini App sign-in.
	TelegramBotToken string
	// TelegramSender, when set, delivers notifications as Telegram messages.
	TelegramSender notifications.Sender
	// UploadDir is where uploaded photos are stored.
	UploadDir string
	// WebAppDir holds the built Mini App, served at "/" when present.
	WebAppDir string
}

// New builds the application: repo → service → handler for each domain.
func New(pool *pgxpool.Pool, cfg Config) (*App, error) {
	worker := matches.NewWorker(pool, cfg.MatchInterval)

	userSvc := users.NewService(users.NewRepository(pool), users.NewRefreshTokenRepository(pool), users.Options{
		JWTSecret:        cfg.JWTSecret,
		Mailer:           cfg.Mailer,
		AppURL:           cfg.AppURL,
		TelegramBotToken: cfg.TelegramBotToken,
	})

	if cfg.UploadDir == "" {
		cfg.UploadDir = "data/uploads"
	}
	store, err := uploads.NewStore(cfg.UploadDir)
	if err != nil {
		return nil, err
	}
	itemSvc := items.NewService(items.NewRepository(pool), worker.Trigger)
	wantSvc := wants.NewService(wants.NewRepository(pool), worker.Trigger)
	matchSvc := matches.NewService(matches.NewRepository(pool))
	ratingSvc := ratings.NewService(ratings.NewRepository(pool))

	handler := router.New(router.Handlers{
		Users:         users.NewHandler(userSvc),
		Items:         items.NewHandler(itemSvc),
		Wants:         wants.NewHandler(wantSvc),
		Matches:       matches.NewHandler(matchSvc),
		Ratings:       ratings.NewHandler(ratingSvc),
		Messages:      messages.NewHandler(messages.NewService(pool)),
		Notifications: notifications.NewHandler(notifications.NewService(pool)),
		Uploads:       uploads.NewHandler(store),
	}, router.Options{
		JWTSecret:     cfg.JWTSecret,
		AuthRateLimit: cfg.AuthRateLimit,
		TrustProxy:    cfg.TrustProxy,
		CORSOrigins:   cfg.CORSOrigins,
		WebAppDir:     cfg.WebAppDir,
	})

	a := &App{Handler: handler, Worker: worker}
	if cfg.TelegramSender != nil {
		a.Dispatcher = notifications.NewDispatcher(pool, cfg.TelegramSender, cfg.AppURL, 5*time.Second)
	}
	return a, nil
}
