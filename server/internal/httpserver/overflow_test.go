package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go-react-shadcn/internal/config"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/security"
	"go-react-shadcn/internal/seed"
)

func TestCorsAllowOriginsDropsLocalhostInProduction(t *testing.T) {
	prod := corsAllowOrigins(config.Config{
		DevMode:    false,
		CORSOrigin: "https://admin.example, https://web.example/",
	})
	want := map[string]bool{"https://admin.example": true, "https://web.example": true}
	if len(prod) != 2 {
		t.Fatalf("prod origins=%v", prod)
	}
	for _, origin := range prod {
		if !want[origin] {
			t.Fatalf("unexpected prod origin %q", origin)
		}
		if origin == "http://localhost:5173" || origin == "http://127.0.0.1:5173" {
			t.Fatal("production still allows localhost")
		}
	}

	dev := corsAllowOrigins(config.Config{DevMode: true, CORSOrigin: "https://admin.example"})
	foundLocal := false
	for _, origin := range dev {
		if origin == "http://localhost:5173" {
			foundLocal = true
		}
	}
	if !foundLocal {
		t.Fatalf("dev should keep localhost, got %v", dev)
	}

	app := testApp(t)
	app.Cfg.DevMode = false
	app.Cfg.CORSOrigin = "https://admin.example"
	app.Router = app.buildRouter()

	preflight := func(origin string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/settings", nil)
		req.Header.Set("Origin", origin)
		req.Header.Set("Access-Control-Request-Method", "GET")
		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)
		return w
	}
	denied := preflight("http://localhost:5173")
	if denied.Header().Get("Access-Control-Allow-Origin") == "http://localhost:5173" {
		t.Fatalf("localhost allowed in production: %d %v", denied.Code, denied.Header())
	}
	allowed := preflight("https://admin.example")
	if allowed.Header().Get("Access-Control-Allow-Origin") != "https://admin.example" {
		t.Fatalf("admin origin: %d %s", allowed.Code, allowed.Header().Get("Access-Control-Allow-Origin"))
	}
	if allowed.Header().Get("Strict-Transport-Security") == "" {
		t.Fatal("expected HSTS in production")
	}

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/settings", nil)
	req.Header.Set("Origin", "https://admin.example")
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "X-Locale, Authorization")
	headerCheck := httptest.NewRecorder()
	app.Router.ServeHTTP(headerCheck, req)
	allow := strings.ToLower(headerCheck.Header().Get("Access-Control-Allow-Headers"))
	if !strings.Contains(allow, "x-locale") {
		t.Fatalf("expected X-Locale in CORS allow headers, got %q", headerCheck.Header().Get("Access-Control-Allow-Headers"))
	}
}

func TestInvalidRoleDataScopeRejected(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	created := doJSON(t, app, http.MethodPost, "/api/v1/roles", admin, map[string]any{
		"name": "Bad scope", "code": "badscope", "dataScope": "everything",
	})
	if created.Code != http.StatusBadRequest || decodeEnv(t, created).ErrorCode != CodeInvalidDataScope {
		t.Fatalf("create: %d %s", created.Code, created.Body.String())
	}
	var viewer models.Role
	if err := app.DB.Where("code = ?", seed.RoleViewer).First(&viewer).Error; err != nil {
		t.Fatal(err)
	}
	upd := doJSON(t, app, http.MethodPut, "/api/v1/roles/"+itoa(viewer.ID), admin, map[string]any{
		"dataScope": "nope",
	})
	if upd.Code != http.StatusBadRequest || decodeEnv(t, upd).ErrorCode != CodeInvalidDataScope {
		t.Fatalf("update: %d %s", upd.Code, upd.Body.String())
	}
}

func TestPrivilegedRoleAssignmentGuard(t *testing.T) {
	app := testApp(t)
	operator := loginOK(t, app, seed.OperatorUsername, seed.OperatorPassword)
	var adminRole models.Role
	if err := app.DB.Where("code = ?", seed.RoleAdmin).First(&adminRole).Error; err != nil {
		t.Fatal(err)
	}
	created := doJSON(t, app, http.MethodPost, "/api/v1/users", operator, map[string]any{
		"username": "priv-op", "password": "Priv-pass1", "status": "active",
		"roleIds": []uint{adminRole.ID},
	})
	if created.Code != http.StatusForbidden || decodeEnv(t, created).ErrorCode != CodePrivilegedRole {
		t.Fatalf("operator create admin: %d %s", created.Code, created.Body.String())
	}

	okUser := doJSON(t, app, http.MethodPost, "/api/v1/users", operator, map[string]any{
		"username": "priv-plain", "password": "Priv-pass1", "status": "active", "department": "ops",
	})
	if okUser.Code != http.StatusOK {
		t.Fatalf("operator create plain: %d %s", okUser.Code, okUser.Body.String())
	}
	var dto userDTO
	if err := json.Unmarshal(decodeEnv(t, okUser).Data, &dto); err != nil {
		t.Fatal(err)
	}

	var perm models.Permission
	if err := app.DB.Where("code = ?", "user:roles").First(&perm).Error; err != nil {
		t.Fatal(err)
	}
	var opRole models.Role
	if err := app.DB.Where("code = ?", seed.RoleOperator).Preload("Permissions").First(&opRole).Error; err != nil {
		t.Fatal(err)
	}
	opRole.Permissions = append(opRole.Permissions, perm)
	if err := app.DB.Model(&opRole).Association("Permissions").Replace(opRole.Permissions); err != nil {
		t.Fatal(err)
	}
	if err := seed.SyncRolePolicies(app.Enforcer, opRole.Code, opRole.Permissions); err != nil {
		t.Fatal(err)
	}

	assign := doJSON(t, app, http.MethodPut, "/api/v1/users/"+itoa(dto.ID)+"/roles", operator, map[string]any{
		"roleIds": []uint{adminRole.ID},
	})
	if assign.Code != http.StatusForbidden || decodeEnv(t, assign).ErrorCode != CodePrivilegedRole {
		t.Fatalf("operator assign admin: %d %s", assign.Code, assign.Body.String())
	}

	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	granted := doJSON(t, app, http.MethodPut, "/api/v1/users/"+itoa(dto.ID)+"/roles", admin, map[string]any{
		"roleIds": []uint{adminRole.ID},
	})
	if granted.Code != http.StatusOK {
		t.Fatalf("admin assign: %d %s", granted.Code, granted.Body.String())
	}
}

func TestDepartmentDeleteAndCycle(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	parent := doJSON(t, app, http.MethodPost, "/api/v1/departments", admin, map[string]any{
		"name": "Overflow Parent", "code": "ov-parent", "status": "active",
	})
	if parent.Code != http.StatusOK {
		t.Fatalf("parent: %d %s", parent.Code, parent.Body.String())
	}
	var parentDept models.Department
	if err := json.Unmarshal(decodeEnv(t, parent).Data, &parentDept); err != nil {
		t.Fatal(err)
	}
	child := doJSON(t, app, http.MethodPost, "/api/v1/departments", admin, map[string]any{
		"name": "Overflow Child", "code": "ov-child", "status": "active", "parentId": parentDept.ID,
	})
	if child.Code != http.StatusOK {
		t.Fatalf("child: %d %s", child.Code, child.Body.String())
	}
	var childDept models.Department
	if err := json.Unmarshal(decodeEnv(t, child).Data, &childDept); err != nil {
		t.Fatal(err)
	}
	cycled := doJSON(t, app, http.MethodPut, "/api/v1/departments/"+itoa(parentDept.ID), admin, map[string]any{
		"name": parentDept.Name, "code": parentDept.Code, "status": "active", "parentId": childDept.ID,
	})
	if cycled.Code != http.StatusBadRequest || decodeEnv(t, cycled).ErrorCode != CodeDeptCycle {
		t.Fatalf("cycle: %d %s", cycled.Code, cycled.Body.String())
	}

	user := doJSON(t, app, http.MethodPost, "/api/v1/users", admin, map[string]any{
		"username": "ov-dept-user", "password": "Dept-pass1", "status": "active",
	})
	if user.Code != http.StatusOK {
		t.Fatalf("user: %d %s", user.Code, user.Body.String())
	}
	var dto userDTO
	if err := json.Unmarshal(decodeEnv(t, user).Data, &dto); err != nil {
		t.Fatal(err)
	}
	if err := app.DB.Table(models.AccountTable(models.UserKindAdmin)).Where("id = ?", dto.ID).Updates(map[string]any{
		"department_id": childDept.ID,
		"department":    childDept.Code,
	}).Error; err != nil {
		t.Fatal(err)
	}
	blocked := doJSON(t, app, http.MethodDelete, "/api/v1/departments/"+itoa(childDept.ID), admin, nil)
	if blocked.Code != http.StatusBadRequest || decodeEnv(t, blocked).ErrorCode != CodeDeptHasUsers {
		t.Fatalf("delete with users: %d %s", blocked.Code, blocked.Body.String())
	}
}

func TestVerifyEmailRateLimited(t *testing.T) {
	app := testApp(t)
	app.VerifyGuard = security.NewIPLimiter(2, time.Minute)
	app.VerifyTokenGuard = security.NewIPLimiter(100, time.Minute)
	first := doJSON(t, app, http.MethodPost, "/api/v1/auth/verify-email", "", map[string]string{"token": "nope"})
	if first.Code != http.StatusBadRequest {
		t.Fatalf("first: %d %s", first.Code, first.Body.String())
	}
	second := doJSON(t, app, http.MethodPost, "/api/v1/auth/verify-email", "", map[string]string{"token": "nope"})
	if second.Code != http.StatusBadRequest {
		t.Fatalf("second: %d %s", second.Code, second.Body.String())
	}
	blocked := doJSON(t, app, http.MethodPost, "/api/v1/auth/verify-email", "", map[string]string{"token": "nope"})
	if blocked.Code != http.StatusTooManyRequests || decodeEnv(t, blocked).ErrorCode != CodeVerifyRateLimited {
		t.Fatalf("limit: %d %s", blocked.Code, blocked.Body.String())
	}
}
