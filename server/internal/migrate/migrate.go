package migrate

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "modernc.org/sqlite"
)

//go:embed sql/*.sql
var files embed.FS

func Up(dbPath string) error {
	m, err := newMigrator(dbPath)
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

func Version(dbPath string) (uint, bool, error) {
	m, err := newMigrator(dbPath)
	if err != nil {
		return 0, false, err
	}
	defer m.Close()
	return m.Version()
}

func newMigrator(dbPath string) (*migrate.Migrate, error) {
	src, err := iofs.New(files, "sql")
	if err != nil {
		return nil, fmt.Errorf("migrate source: %w", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	driver, err := sqlite.WithInstance(db, &sqlite.Config{})
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate driver: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "sqlite", driver)
	if err != nil {
		return nil, fmt.Errorf("migrator: %w", err)
	}
	return m, nil
}
