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
	"github.com/yeabt/lewe/internal/users"
	"github.com/yeabt/lewe/internal/wants"
)

// App bundles the HTTP handler and the background matching worker.
type App struct {
	Handler http.Handler
	Worker  *matches.Worker
}

// Config holds the settings the application needs beyond a database pool.
type Config struct {
	JWTSecret        string
	MatchInterval    time.Duration
	AuthRateLimit    int // per client IP per minute; 0 disables
	TrustProxy       bool
	CORSOrigins      []string
	Mailer           mail.Mailer // nil logs emails instead of sending them
	AppURL           string      // base URL for links in emails
	TelegramBotToken string      // enables /auth/telegram when set
}

// New builds the application: repo → service → handler for each domain.
func New(pool *pgxpool.Pool, cfg Config) *App {
	worker := matches.NewWorker(pool, cfg.MatchInterval)

	userSvc := users.NewService(users.NewRepository(pool), users.NewRefreshTokenRepository(pool), users.Options{
		JWTSecret: cfg.JWTSecret,
		Mailer:    cfg.Mailer,
		AppURL:    cfg.AppURL,

		TelegramBotToken: cfg.TelegramBotToken,
	})
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
	}, router.Options{
		JWTSecret:     cfg.JWTSecret,
		AuthRateLimit: cfg.AuthRateLimit,
		TrustProxy:    cfg.TrustProxy,
		CORSOrigins:   cfg.CORSOrigins,
	})

	return &App{Handler: handler, Worker: worker}
}
