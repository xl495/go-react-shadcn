package httpserver

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"go-react-shadcn/internal/googleid"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/seed"
)

func TestRedactJSONSecretsStripsPasswordsAndTokens(t *testing.T) {
	raw := `{"username":"admin","password":"admin123","idToken":"ya29.abc","nested":{"newPassword":"secret-9"}}`
	got, ok := redactJSONSecrets(raw)
	if !ok {
		t.Fatal("expected json redact")
	}
	if strings.Contains(got, "admin123") || strings.Contains(got, "ya29") || strings.Contains(got, "secret-9") {
		t.Fatalf("secrets leaked: %s", got)
	}
	if !strings.Contains(got, `"username":"admin"`) {
		t.Fatalf("username should remain: %s", got)
	}
	if !strings.Contains(got, "********") {
		t.Fatalf("mask missing: %s", got)
	}
}

func TestRedactRequestBodyAuthNonJSON(t *testing.T) {
	got := redactRequestBody("/api/v1/auth/login", "password=admin123")
	if got != "[redacted]" {
		t.Fatalf("got %q", got)
	}
}

func TestLogoutRevokesJWT(t *testing.T) {
	app := testApp(t)
	token := loginOK(t, app, seed.ViewerUsername, seed.ViewerPassword)
	out := doJSON(t, app, http.MethodPost, "/api/v1/auth/logout", token, map[string]any{})
	if out.Code != http.StatusOK {
		t.Fatalf("logout: %d %s", out.Code, out.Body.String())
	}
	me := doJSON(t, app, http.MethodGet, "/api/v1/auth/me", token, nil)
	if me.Code != http.StatusUnauthorized {
		t.Fatalf("stale token want 401 got %d %s", me.Code, me.Body.String())
	}
}

func TestAdminPasswordChangeRevokesJWT(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	viewerTok := loginOK(t, app, seed.ViewerUsername, seed.ViewerPassword)

	var viewer models.User
	if err := app.DB.Where("username = ?", seed.ViewerUsername).First(&viewer).Error; err != nil {
		t.Fatal(err)
	}
	w := doJSON(t, app, http.MethodPut, "/api/v1/users/"+strconv.FormatUint(uint64(viewer.ID), 10), admin, map[string]any{
		"password": "viewer-pass-9",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("update: %d %s", w.Code, w.Body.String())
	}
	me := doJSON(t, app, http.MethodGet, "/api/v1/auth/me", viewerTok, nil)
	if me.Code != http.StatusUnauthorized {
		t.Fatalf("stale token want 401 got %d %s", me.Code, me.Body.String())
	}
}

func TestUserListOmitsNestedPermissions(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	users := decodeUserPage(t, doJSON(t, app, http.MethodGet, "/api/v1/users?pageSize=50", admin, nil))
	if len(users) == 0 {
		t.Fatal("expected users")
	}
	for _, u := range users {
		if len(u.PermissionCodes) > 0 {
			t.Fatalf("list permissionCodes should be empty, user=%s codes=%v", u.Username, u.PermissionCodes)
		}
		for _, r := range u.Roles {
			if len(r.Permissions) > 0 {
				t.Fatalf("list nested permissions should be empty, role=%s", r.Code)
			}
		}
	}
}

func TestRoleListUsesPermissionIDsNotObjects(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	roles := decodeRolePage(t, doJSON(t, app, http.MethodGet, "/api/v1/roles?pageSize=50", admin, nil))
	var adminRole roleDTO
	for _, r := range roles {
		if r.Code == seed.RoleAdmin {
			adminRole = r
			break
		}
	}
	if adminRole.ID == 0 {
		t.Fatal("admin role missing")
	}
	if len(adminRole.Permissions) > 0 {
		t.Fatal("list should omit permission objects")
	}
	if len(adminRole.PermissionIDs) == 0 {
		t.Fatal("list should include permissionIds")
	}
	got := doJSON(t, app, http.MethodGet, "/api/v1/roles/"+formatUint(adminRole.ID), admin, nil)
	if got.Code != http.StatusOK {
		t.Fatalf("get role: %d %s", got.Code, got.Body.String())
	}
	if strings.Contains(got.Body.String(), `"permissions":[`) && strings.Contains(got.Body.String(), `"path":`) {
		t.Fatal("role detail should not embed permission tree")
	}
	if !strings.Contains(got.Body.String(), `"permissionIds"`) {
		t.Fatal("role detail missing permissionIds")
	}
}

func TestAPIEnvelopeJSONNotGin404(t *testing.T) {
	app := testApp(t)
	missing := doJSON(t, app, http.MethodGet, "/api/v1/does-not-exist", "", nil)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("unknown route: %d %s", missing.Code, missing.Body.String())
	}
	if strings.Contains(missing.Body.String(), "404 page not found") {
		t.Fatalf("want json envelope, got gin 404: %s", missing.Body.String())
	}
	env := decodeEnv(t, missing)
	if env.Code != CodeFail || env.ErrorCode != CodeRouteNotFound || env.Message == "" {
		t.Fatalf("envelope=%+v body=%s", env, missing.Body.String())
	}

	unauth := doJSON(t, app, http.MethodGet, "/api/v1/roles/1", "", nil)
	if unauth.Code != http.StatusUnauthorized {
		t.Fatalf("get role without token: %d %s", unauth.Code, unauth.Body.String())
	}
	authEnv := decodeEnv(t, unauth)
	if authEnv.Code != CodeFail || authEnv.Message == "" {
		t.Fatalf("unauth envelope=%+v body=%s", authEnv, unauth.Body.String())
	}
}

func TestGoogleAdminClientRegisterCreatesWebUser(t *testing.T) {
	app := testApp(t)
	setCfg(t, app, "auth.google_enabled", "1")
	setCfg(t, app, "auth.google_register_enabled", "1")
	setCfg(t, app, "auth.google_client_id", "client-1")
	app.GoogleVerify = stubGoogle{ident: googleid.Identity{
		Subject: "gid-new-admin-reg", Email: "selfreg@example.com", EmailVerified: true, Name: "Self",
	}}
	w := doJSON(t, app, http.MethodPost, "/api/v1/auth/google", "", map[string]string{
		"idToken": "tok", "client": "admin",
	})
	if w.Code != http.StatusForbidden || decodeEnv(t, w).ErrorCode != CodeGoogleRegisterDisabled {
		t.Fatalf("want register disabled, got %d %s", w.Code, w.Body.String())
	}
	var n int64
	_ = models.Accounts(app.DB, models.UserKindAdmin).Where("lower(email) = ?", "selfreg@example.com").Count(&n).Error
	if n != 0 {
		t.Fatal("google admin client must not create admin_user")
	}
	_ = models.Accounts(app.DB, models.UserKindWeb).Where("lower(email) = ?", "selfreg@example.com").Count(&n).Error
	if n != 0 {
		t.Fatal("google admin client must not create web_user")
	}
}

func TestCreateUserRejectsDuplicateEmail(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	w := doJSON(t, app, http.MethodPost, "/api/v1/users", admin, map[string]any{
		"username": "dupmail",
		"password": "dupmail12",
		"email":    "admin@latch.local",
		"status":   "active",
	})
	if w.Code != http.StatusConflict || decodeEnv(t, w).ErrorCode != CodeEmailExists {
		t.Fatalf("got %d %s", w.Code, w.Body.String())
	}
}
