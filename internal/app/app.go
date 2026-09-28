// Package app wires repositories, services, handlers and the router together.
package app

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yeabt/lewe/internal/items"
	"github.com/yeabt/lewe/internal/matches"
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

// New builds the application: repo → service → handler for each domain.
func New(pool *pgxpool.Pool, jwtSecret string, matchInterval time.Duration) *App {
	worker := matches.NewWorker(pool, matchInterval)

	userSvc := users.NewService(users.NewRepository(pool), users.NewRefreshTokenRepository(pool), jwtSecret)
	itemSvc := items.NewService(items.NewRepository(pool), worker.Trigger)
	wantSvc := wants.NewService(wants.NewRepository(pool), worker.Trigger)
	matchSvc := matches.NewService(matches.NewRepository(pool))
	ratingSvc := ratings.NewService(ratings.NewRepository(pool))

	handler := router.New(router.Handlers{
		Users:   users.NewHandler(userSvc),
		Items:   items.NewHandler(itemSvc),
		Wants:   wants.NewHandler(wantSvc),
		Matches: matches.NewHandler(matchSvc),
		Ratings: ratings.NewHandler(ratingSvc),
	}, jwtSecret)

	return &App{Handler: handler, Worker: worker}
}
