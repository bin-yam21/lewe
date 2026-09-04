package db

import (
	"errors"
	"log"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// migrateURL rewrites a standard connection string to the scheme golang-migrate's
// pgx/v5 driver registers itself under ("pgx5"). pgxpool accepts postgres:// and
// postgresql://, but migrate resolves its driver from the URL scheme alone and
// would otherwise fail with "unknown driver postgres".
func migrateURL(databaseURL string) string {
	for _, scheme := range []string{"postgres://", "postgresql://"} {
		if rest, ok := strings.CutPrefix(databaseURL, scheme); ok {
			return "pgx5://" + rest
		}
	}
	return databaseURL
}

// RunMigrations applies all pending up-migrations.
// migrationsPath should be a file:// URL pointing to the migrations directory.
func RunMigrations(databaseURL, migrationsPath string) {
	m, err := migrate.New(migrationsPath, migrateURL(databaseURL))
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Migrations applied successfully")
}
