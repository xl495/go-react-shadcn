package migrate

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"

	_ "github.com/glebarez/go-sqlite"
)

//go:embed sql/*.sql
var files embed.FS

const versionTable = "schema_migrations"

func Up(dbPath string) error {
	db, err := openDB(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := ensureVersionTable(db); err != nil {
		return err
	}
	current, dirty, err := readVersion(db)
	if err != nil {
		return err
	}
	if dirty {
		return fmt.Errorf("migrate: database is dirty at version %d", current)
	}
	steps, err := listUpMigrations()
	if err != nil {
		return err
	}
	for _, step := range steps {
		if current >= int(step.version) {
			continue
		}
		body, err := files.ReadFile(step.path)
		if err != nil {
			return fmt.Errorf("read %s: %w", step.path, err)
		}
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(body)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migrate up %d: %w", step.version, err)
		}
		if err := writeVersion(tx, int(step.version), false); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		current = int(step.version)
	}
	return nil
}

func Version(dbPath string) (uint, bool, error) {
	db, err := openDB(dbPath)
	if err != nil {
		return 0, false, err
	}
	defer db.Close()
	if err := ensureVersionTable(db); err != nil {
		return 0, false, err
	}
	version, dirty, err := readVersion(db)
	if err != nil {
		return 0, false, err
	}
	if version < 0 {
		return 0, dirty, nil
	}
	return uint(version), dirty, nil
}

func openDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	return db, nil
}

func ensureVersionTable(db *sql.DB) error {
	_, err := db.Exec(fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (version uint64, dirty bool);
CREATE UNIQUE INDEX IF NOT EXISTS version_unique ON %s (version);
`, versionTable, versionTable))
	if err != nil {
		return fmt.Errorf("ensure version table: %w", err)
	}
	return nil
}

func readVersion(db *sql.DB) (int, bool, error) {
	var version int
	var dirty bool
	err := db.QueryRow(fmt.Sprintf("SELECT version, dirty FROM %s LIMIT 1", versionTable)).Scan(&version, &dirty)
	if err == sql.ErrNoRows {
		return -1, false, nil
	}
	if err != nil {
		return -1, false, err
	}
	return version, dirty, nil
}

func writeVersion(tx *sql.Tx, version int, dirty bool) error {
	if _, err := tx.Exec(fmt.Sprintf("DELETE FROM %s", versionTable)); err != nil {
		return err
	}
	_, err := tx.Exec(fmt.Sprintf("INSERT INTO %s (version, dirty) VALUES (?, ?)", versionTable), version, dirty)
	return err
}

type migrationFile struct {
	version uint
	path    string
}

func listUpMigrations() ([]migrationFile, error) {
	entries, err := fs.ReadDir(files, "sql")
	if err != nil {
		return nil, err
	}
	out := make([]migrationFile, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		n, err := strconv.ParseUint(strings.SplitN(name, "_", 2)[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("migration name %s: %w", name, err)
		}
		out = append(out, migrationFile{version: uint(n), path: path.Join("sql", name)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	return out, nil
}
