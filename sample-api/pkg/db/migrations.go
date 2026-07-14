package db

import (
	"embed"
	"errors"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// RunMigrations applies all pending up migrations bundled in the binary.
// It is a no-op when the database is already at the latest version.
func RunMigrations(databaseURL string) error {
	d, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		return err
	}
	// golang-migrate's database driver is registered under the "postgres"
	// scheme; config URLs commonly use "postgresql://". Normalize it.
	if rest, ok := strings.CutPrefix(databaseURL, "postgresql://"); ok {
		databaseURL = "postgres://" + rest
	}
	m, err := migrate.NewWithSourceInstance("iofs", d, databaseURL)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
