package db

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations applies any pending migrations from the given directory.
// It is idempotent: if the database is already at the latest version, it
// returns nil.
func RunMigrations(dsn, migrationsPath string) error {
	src := "file://" + migrationsPath
	m, err := migrate.New(src, dsn)
	if err != nil {
		return fmt.Errorf("migrate: init: %w", err)
	}
	defer func() {
		_, _ = m.Close()
	}()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate: up: %w", err)
	}
	return nil
}
