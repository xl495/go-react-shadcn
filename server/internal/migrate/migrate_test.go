package migrate

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestUpCreatesSchemaAndIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.db")
	if err := Up(path); err != nil {
		t.Fatalf("first up: %v", err)
	}
	if err := Up(path); err != nil {
		t.Fatalf("second up: %v", err)
	}

	version, dirty, err := Version(path)
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	if dirty {
		t.Fatal("expected clean migration state")
	}
	if version != 2 {
		t.Fatalf("version=%d, want 2", version)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	required := []string{"users", "roles", "permissions", "op_logs", "login_logs", "api_logs", "departments", "casbin_rule", "schema_migrations"}
	for _, name := range required {
		var got string
		err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, name).Scan(&got)
		if err != nil {
			t.Fatalf("table %s missing: %v", name, err)
		}
		if got != name {
			t.Fatalf("table name=%q want %q", got, name)
		}
	}

	if _, err := db.Exec(`INSERT INTO op_logs (username, module, action, method, path, status, ip, latency_ms, detail) VALUES (?,?,?,?,?,?,?,?,?)`,
		"admin", "auth", "login", "POST", "/api/v1/auth/login", 200, "127.0.0.1", 3, "ok:admin"); err != nil {
		t.Fatalf("insert op_logs: %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM op_logs WHERE username=? AND action=?`, "admin", "login").Scan(&count); err != nil {
		t.Fatalf("count op_logs: %v", err)
	}
	if count != 1 {
		t.Fatalf("op_logs count=%d want 1", count)
	}
}
