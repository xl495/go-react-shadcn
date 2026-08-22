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
	if version != 1 {
		t.Fatalf("version=%d, want 1", version)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	required := []string{"admin_user", "web_user", "web_user_roles", "roles", "permissions", "op_logs", "login_logs", "api_logs", "departments", "casbin_rule", "schema_migrations", "password_reset_tokens", "mail_jobs", "mail_campaigns", "nav_menu", "auth_sessions", "user_import_jobs", "captcha_challenges", "notifications", "totp_recovery_codes"}
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

	var idx string
	if err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='index' AND name='idx_auth_sessions_online'`).Scan(&idx); err != nil {
		t.Fatalf("online session index missing: %v", err)
	}
	var col int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('web_user') WHERE name='must_set_password'`).Scan(&col); err != nil || col != 1 {
		t.Fatalf("web_user.must_set_password missing: n=%d err=%v", col, err)
	}
	for _, leftover := range []string{"users", "admin_menus", "web_menus"} {
		var name string
		err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, leftover).Scan(&name)
		if err == nil {
			t.Fatalf("legacy table %s should not exist", leftover)
		}
		if err != sql.ErrNoRows {
			t.Fatalf("legacy table %s: %v", leftover, err)
		}
	}
}

func TestSplitSQLKeepsTriggerBodies(t *testing.T) {
	body, err := files.ReadFile("sql/000001_init.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	stmts := splitSQL(string(body))
	var found bool
	for _, stmt := range stmts {
		if strings.Contains(stmt, "CREATE TRIGGER") && strings.Contains(stmt, "END") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("trigger split lost END in %d statements", len(stmts))
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

func TestFoldedLegacyVersionStampsToOne(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`
CREATE TABLE schema_migrations (version uint64, dirty bool);
CREATE UNIQUE INDEX version_unique ON schema_migrations (version);
INSERT INTO schema_migrations (version, dirty) VALUES (24, 0);
CREATE TABLE admin_user (id INTEGER);
`); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	if err := Up(path); err != nil {
		t.Fatalf("fold v24: %v", err)
	}
	version, dirty, err := Version(path)
	if err != nil {
		t.Fatal(err)
	}
	if dirty || version != 1 {
		t.Fatalf("version=%d dirty=%v, want 1", version, dirty)
	}
}

func TestIncompleteLegacyVersionRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
CREATE TABLE schema_migrations (version uint64, dirty bool);
INSERT INTO schema_migrations (version, dirty) VALUES (10, 0);
`); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	err = Up(path)
	if err == nil || !strings.Contains(err.Error(), "version 10") {
		t.Fatalf("want version 10 error, got %v", err)
	}
}
