package migrate

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/glebarez/go-sqlite"
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
	if version != 20 {
		t.Fatalf("version=%d, want 20", version)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	required := []string{"users", "admin_user", "web_user", "web_user_roles", "roles", "permissions", "op_logs", "login_logs", "api_logs", "departments", "casbin_rule", "schema_migrations", "password_reset_tokens", "mail_jobs", "mail_campaigns", "admin_menus", "web_menus", "nav_menu", "auth_sessions", "user_import_jobs", "captcha_challenges", "notifications", "totp_recovery_codes"}
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

func TestSplitSQLKeepsTriggerBodies(t *testing.T) {
	body, err := files.ReadFile("sql/000013_user_fts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	stmts := splitSQL(string(body))
	if len(stmts) != 10 {
		t.Fatalf("stmts=%d want 10: %#v", len(stmts), stmts)
	}
	if !strings.Contains(stmts[2], "END") {
		t.Fatalf("trigger split lost END: %s", stmts[2])
	}
}

func TestExecMigrationSkipsDuplicateColumn(t *testing.T) {
	db, err := sql.Open("sqlite", "file:dupcol?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`CREATE TABLE users (id INTEGER, timezone TEXT)`); err != nil {
		t.Fatal(err)
	}
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })
	sqlBody := `
ALTER TABLE users ADD COLUMN timezone TEXT NOT NULL DEFAULT 'Asia/Shanghai';
ALTER TABLE users ADD COLUMN marketing_opt_in INTEGER NOT NULL DEFAULT 1;
`
	if err := execMigration(tx, sqlBody); err != nil {
		t.Fatalf("exec: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('users') WHERE name='marketing_opt_in'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("marketing_opt_in missing")
	}
}
