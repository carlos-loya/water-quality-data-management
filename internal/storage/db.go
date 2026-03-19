package storage

import (
	"context"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect creates a connection pool to the database.
func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return pool, nil
}

// Migrate runs all pending up migrations. It is safe to call on every startup;
// if the database is already up to date it returns nil.
func Migrate(databaseURL, sourcePath string) error {
	// golang-migrate uses the pgx5 scheme for the pgx/v5 driver.
	m, err := migrate.New(sourcePath, pgxMigrateURL(databaseURL))
	if err != nil {
		return fmt.Errorf("init migrate: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

// pgxMigrateURL converts a postgres:// URL to pgx5:// for golang-migrate.
func pgxMigrateURL(url string) string {
	if len(url) > 11 && url[:11] == "postgres://" {
		return "pgx5://" + url[11:]
	}
	if len(url) > 14 && url[:14] == "postgresql://" {
		return "pgx5://" + url[14:]
	}
	return url
}
