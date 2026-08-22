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

// foldedLegacyMax is the last incremental version folded into 000001_init.
// Databases already at this version are stamped to 1 without re-running SQL.
const foldedLegacyMax = 24

func Up(dbPath string) error {
	db, err := openDB(dbPath)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
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
	latest := 0
	if n := len(steps); n > 0 {
		latest = steps[n-1].version
	}
	if current > latest {
		if current != foldedLegacyMax {
			return fmt.Errorf("migrate: database is at version %d; folded schema requires version %d or a new database", current, foldedLegacyMax)
		}
		if err := stampVersion(db, 1); err != nil {
			return err
		}
		current = 1
	}
	for _, step := range steps {
		if current >= step.version {
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
		if err := execMigration(tx, string(body)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migrate up %d: %w", step.version, err)
		}
		if err := writeVersion(tx, step.version, false); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		current = step.version
	}
	return nil
}

func Version(dbPath string) (uint, bool, error) {
	db, err := openDB(dbPath)
	if err != nil {
		return 0, false, err
	}
	defer func() { _ = db.Close() }()
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

func stampVersion(db *sql.DB, version int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if err := writeVersion(tx, version, false); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func writeVersion(tx *sql.Tx, version int, dirty bool) error {
	if _, err := tx.Exec(fmt.Sprintf("DELETE FROM %s", versionTable)); err != nil {
		return err
	}
	_, err := tx.Exec(fmt.Sprintf("INSERT INTO %s (version, dirty) VALUES (?, ?)", versionTable), version, dirty)
	return err
}

type migrationFile struct {
	version int
	path    string
}

func execMigration(tx *sql.Tx, body string) error {
	for _, stmt := range splitSQL(body) {
		if _, err := tx.Exec(stmt); err != nil {
			if ignorableMigrationErr(err) {
				continue
			}
			return err
		}
	}
	return nil
}

func ignorableMigrationErr(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate column name") || strings.Contains(msg, "already exists")
}

func splitSQL(script string) []string {
	runes := []rune(strings.TrimSpace(script))
	if len(runes) == 0 {
		return nil
	}
	var out []string
	var b strings.Builder
	depth := 0
	flush := func() {
		stmt := strings.TrimSpace(b.String())
		b.Reset()
		if stmt != "" {
			out = append(out, stmt)
		}
	}
	for i := 0; i < len(runes); {
		if runes[i] == '-' && i+1 < len(runes) && runes[i+1] == '-' {
			for i < len(runes) && runes[i] != '\n' {
				i++
			}
			continue
		}
		if runes[i] == '/' && i+1 < len(runes) && runes[i+1] == '*' {
			i += 2
			for i+1 < len(runes) && (runes[i] != '*' || runes[i+1] != '/') {
				i++
			}
			if i+1 < len(runes) {
				i += 2
			}
			continue
		}
		if runes[i] == '\'' {
			b.WriteRune(runes[i])
			i++
			for i < len(runes) {
				b.WriteRune(runes[i])
				if runes[i] == '\'' {
					if i+1 < len(runes) && runes[i+1] == '\'' {
						b.WriteRune(runes[i+1])
						i += 2
						continue
					}
					i++
					break
				}
				i++
			}
			continue
		}
		if ident, ok := readIdent(runes, i); ok {
			switch strings.ToUpper(ident) {
			case "BEGIN":
				depth++
			case "END":
				if depth > 0 {
					depth--
				}
			}
			b.WriteString(ident)
			i += len([]rune(ident))
			continue
		}
		if runes[i] == ';' && depth == 0 {
			flush()
			i++
			continue
		}
		b.WriteRune(runes[i])
		i++
	}
	flush()
	return out
}

func readIdent(runes []rune, i int) (string, bool) {
	if i >= len(runes) {
		return "", false
	}
	r := runes[i]
	if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') && r != '_' {
		return "", false
	}
	j := i + 1
	for j < len(runes) {
		r = runes[j]
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			j++
			continue
		}
		break
	}
	return string(runes[i:j]), true
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
		n, err := strconv.Atoi(strings.SplitN(name, "_", 2)[0])
		if err != nil {
			return nil, fmt.Errorf("migration name %s: %w", name, err)
		}
		if n < 0 {
			return nil, fmt.Errorf("migration name %s: negative version", name)
		}
		out = append(out, migrationFile{version: n, path: path.Join("sql", name)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	return out, nil
}
