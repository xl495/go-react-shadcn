package httpserver

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"go-react-shadcn/internal/config"
	"go-react-shadcn/internal/migrate"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/seed"
	"go-react-shadcn/internal/store"
)

func TestSecurityHeadersPresent(t *testing.T) {
	app := testApp(t)
	w := doJSON(t, app, http.MethodGet, "/health", "", nil)
	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("nosniff missing: %v", w.Header())
	}
	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("frame options missing: %v", w.Header())
	}
	if csp := w.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "frame-ancestors 'none'") {
		t.Fatalf("csp missing: %v", w.Header())
	}
}

func TestMetricsTokenRequiredWhenConfigured(t *testing.T) {
	app := testApp(t)
	app.Cfg.MetricsToken = "metrics-secret"
	denied := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	app.Router.ServeHTTP(denied, req)
	if denied.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 got %d", denied.Code)
	}
	okw := httptest.NewRecorder()
	okreq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	okreq.Header.Set("X-Metrics-Token", "metrics-secret")
	app.Router.ServeHTTP(okw, okreq)
	if okw.Code != http.StatusOK {
		t.Fatalf("token metrics: %d %s", okw.Code, okw.Body.String())
	}
	if !strings.Contains(okw.Body.String(), "latch_http_requests_total") {
		t.Fatalf("metrics body: %s", okw.Body.String())
	}
	if !strings.Contains(okw.Body.String(), "latch_audit_dropped_total") {
		t.Fatalf("audit metrics missing: %s", okw.Body.String())
	}
}

func TestJSONBodySizeLimit(t *testing.T) {
	app := testApp(t)
	raw := []byte(`{"username":"` + strings.Repeat("a", int(jsonBodyLimit)+32) + `","password":"x"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest && w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body want 400/413 got %d %s", w.Code, w.Body.String())
	}
}

func TestProductionRejectsSeedPasswordLogin(t *testing.T) {
	app := testApp(t)
	app.Cfg.DevMode = false
	id, ans, _ := issueCaptcha(t, app)
	w := login(t, app, seed.AdminUsername, seed.AdminPassword, id, ans)
	if w.Code != http.StatusForbidden || decodeEnv(t, w).ErrorCode != CodeSeedPassword {
		t.Fatalf("want seed password forbidden, got %d %s", w.Code, w.Body.String())
	}
}

func TestDataScopeBlocksUserGetAndUpdate(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)

	var opRole roleDTO
	for _, r := range decodeRolePage(t, doJSON(t, app, http.MethodGet, "/api/v1/roles?pageSize=50", admin, nil)) {
		if r.Code == seed.RoleOperator {
			opRole = r
			break
		}
	}
	if opRole.ID == 0 {
		t.Fatal("operator role missing")
	}
	perms := decodePage[permissionDTO](t, doJSON(t, app, http.MethodGet, "/api/v1/permissions?pageSize=200", admin, nil)).Items
	var detailID, updateID uint
	for _, p := range perms {
		switch p.Code {
		case "user:detail":
			detailID = p.ID
		case "user:update":
			updateID = p.ID
		}
	}
	if detailID == 0 || updateID == 0 {
		t.Fatal("missing user:detail/update")
	}
	ids := append(append([]uint{}, opRole.PermissionIDs...), detailID, updateID)
	if w := doJSON(t, app, http.MethodPut, "/api/v1/roles/"+strconv.FormatUint(uint64(opRole.ID), 10)+"/permissions", admin, map[string]any{
		"permissionIds": ids,
	}); w.Code != http.StatusOK {
		t.Fatalf("grant perms: %d %s", w.Code, w.Body.String())
	}

	var adminUser, opUser models.User
	if err := app.DB.Where("username = ?", seed.AdminUsername).First(&adminUser).Error; err != nil {
		t.Fatal(err)
	}
	if err := app.DB.Where("username = ?", seed.OperatorUsername).First(&opUser).Error; err != nil {
		t.Fatal(err)
	}
	opTok := loginOK(t, app, seed.OperatorUsername, seed.OperatorPassword)
	hidden := doJSON(t, app, http.MethodGet, "/api/v1/users/"+strconv.FormatUint(uint64(adminUser.ID), 10), opTok, nil)
	if hidden.Code != http.StatusNotFound {
		t.Fatalf("cross-dept get want 404 got %d %s", hidden.Code, hidden.Body.String())
	}
	self := doJSON(t, app, http.MethodGet, "/api/v1/users/"+strconv.FormatUint(uint64(opUser.ID), 10), opTok, nil)
	if self.Code != http.StatusOK {
		t.Fatalf("self get: %d %s", self.Code, self.Body.String())
	}
	upd := doJSON(t, app, http.MethodPut, "/api/v1/users/"+strconv.FormatUint(uint64(adminUser.ID), 10), opTok, map[string]any{
		"nickname": "nope",
	})
	if upd.Code != http.StatusNotFound {
		t.Fatalf("cross-dept update want 404 got %d %s", upd.Code, upd.Body.String())
	}
}

func TestProductionUsesSQLMigrateAndBootstrapAdmin(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.db")
	if err := migrate.Up(path); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BOOTSTRAP_ADMIN_PASSWORD", "Bootstrap1x")
	db, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		DevMode:         false,
		DatabasePath:    path,
		JWTSecret:       "abcdefghijklmnopqrstuvwxyz012345",
		MailUnsubSecret: "zyxwvutsrqponmlkjihgfedcba543210",
		JWTTTL:          time.Hour,
		CORSOrigin:      "http://localhost:5173",
		UploadDir:       filepath.Join(dir, "uploads"),
		APILogEnabled:   true,
		APILogSample:    1,
	}
	app, err := New(cfg, db)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	t.Cleanup(app.Close)

	var n int64
	_ = app.DB.Model(&models.User{}).Where("username = ?", seed.ViewerUsername).Count(&n).Error
	if n != 0 {
		t.Fatal("production should not seed demo viewer")
	}
	tok := loginOK(t, app, seed.AdminUsername, "Bootstrap1x")
	me := doJSON(t, app, http.MethodGet, "/api/v1/auth/me", tok, nil)
	if me.Code != http.StatusOK {
		t.Fatalf("me: %d %s", me.Code, me.Body.String())
	}
}
