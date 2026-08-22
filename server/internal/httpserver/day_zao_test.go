package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go-react-shadcn/internal/captcha"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/seed"
)

func TestResetPasswordRevokesSessions(t *testing.T) {
	app := testApp(t)
	tok := loginOK(t, app, seed.ViewerUsername, seed.ViewerPassword)
	mem := useMemoryMail(t, app)

	id, answer, _ := issueCaptcha(t, app)
	forgot := doJSON(t, app, http.MethodPost, "/api/v1/auth/forgot-password", "", map[string]string{
		"email": "viewer@latch.local", "captchaId": id, "captchaCode": answer,
	})
	if forgot.Code != http.StatusOK {
		t.Fatalf("forgot: %d %s", forgot.Code, forgot.Body.String())
	}
	msg, ok := mem.Last()
	if !ok {
		t.Fatal("expected reset mail")
	}
	token := tokenFromResetMail(t, msg.Body)
	reset := doJSON(t, app, http.MethodPost, "/api/v1/auth/reset-password", "", map[string]string{
		"token": token, "newPassword": "ViewerReset9",
	})
	if reset.Code != http.StatusOK {
		t.Fatalf("reset: %d %s", reset.Code, reset.Body.String())
	}
	stale := doJSON(t, app, http.MethodGet, "/api/v1/auth/me", tok, nil)
	if stale.Code != http.StatusUnauthorized {
		t.Fatalf("old jwt after reset: %d %s", stale.Code, stale.Body.String())
	}
	if loginOK(t, app, seed.ViewerUsername, "ViewerReset9") == "" {
		t.Fatal("new password should work")
	}
}

func TestForgotPasswordIgnoresUntrustedOrigin(t *testing.T) {
	app := testApp(t)
	mem := useMemoryMail(t, app)

	// CORS would reject Origin: http://evil.example before the handler.
	// The mail helper must still ignore it when building the link.
	if link, ok := app.mailPublicLink("http://127.0.0.1:5173", "http://evil.example", "http://127.0.0.1:5174", "/reset-password?token=abc"); !ok || strings.Contains(link, "evil.example") {
		t.Fatalf("untrusted origin must not win: %q ok=%v", link, ok)
	}
	if link, ok := app.mailPublicLink("http://127.0.0.1:5173", "http://127.0.0.1:5174", "http://127.0.0.1:5173", "/verify-email?token=abc"); !ok || !strings.Contains(link, "http://127.0.0.1:5174/verify-email") {
		t.Fatalf("allowlisted origin should win: %q ok=%v", link, ok)
	}
	if link, ok := app.mailPublicLink("", "http://evil.example", "http://127.0.0.1:5173", "/reset-password?token=abc"); !ok || strings.Contains(link, "evil.example") {
		t.Fatalf("untrusted origin must not win: %q ok=%v", link, ok)
	}

	id, answer, _ := issueCaptcha(t, app)
	forgot := doJSON(t, app, http.MethodPost, "/api/v1/auth/forgot-password", "", map[string]string{
		"email": "Admin@latch.local", "captchaId": id, "captchaCode": answer,
	})
	if forgot.Code != http.StatusOK {
		t.Fatalf("forgot: %d %s", forgot.Code, forgot.Body.String())
	}
	msg, ok := mem.Last()
	if !ok {
		t.Fatal("expected reset mail")
	}
	if !strings.Contains(msg.Body, "http://127.0.0.1:5173/reset-password?token=") {
		t.Fatalf("expected configured base url, got %s", msg.Body)
	}

	setCfg(t, app, "mail.reset_base_url", "")
	id, answer, _ = issueCaptcha(t, app)
	allow := doJSONOrigin(t, app, "/api/v1/auth/forgot-password", map[string]string{
		"email": "Admin@latch.local", "captchaId": id, "captchaCode": answer,
	}, "http://localhost:5173")
	if allow.Code != http.StatusOK {
		t.Fatalf("allowlisted origin: %d %s", allow.Code, allow.Body.String())
	}
	msg, ok = mem.Last()
	if !ok || !strings.Contains(msg.Body, "http://localhost:5173/reset-password?token=") {
		t.Fatalf("dev allowlisted origin: %+v", msg)
	}

	app.Cfg.DevMode = false
	n := mem.Count()
	id, answer, _ = issueCaptcha(t, app)
	prod := doJSONOrigin(t, app, "/api/v1/auth/forgot-password", map[string]string{
		"email": "Admin@latch.local", "captchaId": id, "captchaCode": answer,
	}, "http://localhost:5173")
	if prod.Code != http.StatusOK {
		t.Fatalf("prod forgot: %d %s", prod.Code, prod.Body.String())
	}
	if mem.Count() != n {
		t.Fatal("production must not send reset mail without mail.reset_base_url")
	}
}

func TestProductionCaptchaCannotBeNone(t *testing.T) {
	app := testApp(t)
	setCfg(t, app, "auth.captcha_provider", "none")
	app.Cfg.DevMode = false
	app.Captcha = captcha.New(app.DB, false)

	settings := doJSON(t, app, http.MethodGet, "/api/v1/auth/settings", "", nil)
	if settings.Code != http.StatusOK {
		t.Fatalf("settings: %d %s", settings.Code, settings.Body.String())
	}
	if strings.Contains(settings.Body.String(), `"captchaProvider":"none"`) {
		t.Fatalf("production advertised none: %s", settings.Body.String())
	}

	login := doJSON(t, app, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"username": seed.AdminUsername, "password": seed.AdminPassword,
	})
	if login.Code != http.StatusBadRequest {
		t.Fatalf("login without captcha: %d %s", login.Code, login.Body.String())
	}

	issued := doJSON(t, app, http.MethodGet, "/api/v1/auth/captcha", "", nil)
	if issued.Code != http.StatusOK {
		t.Fatalf("captcha: %d", issued.Code)
	}
	if strings.Contains(issued.Body.String(), `"answer"`) {
		t.Fatalf("captcha debug leaked in production: %s", issued.Body.String())
	}

	app.Cfg.DevMode = true
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	var row models.SysConfig
	if err := app.DB.Where(`"key" = ?`, "auth.captcha_provider").First(&row).Error; err != nil {
		t.Fatal(err)
	}
	app.Cfg.DevMode = false
	denied := doJSON(t, app, http.MethodPut, "/api/v1/configs/"+itoa(row.ID), admin, map[string]string{
		"value": "none", "name": row.Name, "group": "auth",
	})
	if denied.Code != http.StatusBadRequest {
		t.Fatalf("save none in production: %d %s", denied.Code, denied.Body.String())
	}
}

func TestImportJobScopedToActor(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	csvBody := "username,password,nickname,email,phone,status,department,kind\nidoruser,idoruser12,IDOR,idor@example.com,,active,,admin\n"
	posted := doMultipart(t, app, "/api/v1/users/import?kind=admin", admin, "file", "users.csv", []byte(csvBody))
	if posted.Code != http.StatusOK {
		t.Fatalf("import: %d %s", posted.Code, posted.Body.String())
	}
	var job models.UserImportJob
	if err := json.Unmarshal(decodeEnv(t, posted).Data, &job); err != nil || job.ID == 0 {
		t.Fatalf("job: %s", posted.Body.String())
	}
	for i := 0; i < 40; i++ {
		got := doJSON(t, app, http.MethodGet, "/api/v1/users/import-jobs/"+formatUint(job.ID), admin, nil)
		if got.Code != http.StatusOK {
			t.Fatalf("owner get: %d %s", got.Code, got.Body.String())
		}
		var row models.UserImportJob
		if err := json.Unmarshal(decodeEnv(t, got).Data, &row); err != nil {
			t.Fatal(err)
		}
		if row.Status == "done" || row.Status == "failed" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	operator := loginOK(t, app, seed.OperatorUsername, seed.OperatorPassword)
	peek := doJSON(t, app, http.MethodGet, "/api/v1/users/import-jobs/"+formatUint(job.ID), operator, nil)
	if peek.Code != http.StatusNotFound {
		t.Fatalf("operator idor: %d %s", peek.Code, peek.Body.String())
	}
}

func TestAvatarPathTraversalRejected(t *testing.T) {
	app := testApp(t)
	root := app.Cfg.UploadDir
	if err := os.MkdirAll(filepath.Join(root, "avatars"), 0o750); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(filepath.Dir(root), "secret.txt")
	if err := os.WriteFile(secret, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	removeUploadedFile(root, "/uploads/../secret.txt")
	removeUploadedFile(root, "/uploads/avatars/../../secret.txt")
	if _, err := os.Stat(secret); err != nil {
		t.Fatalf("traversal deleted file outside upload dir: %v", err)
	}

	safe := filepath.Join(root, "avatars", "ok.png")
	if err := os.WriteFile(safe, []byte("img"), 0o600); err != nil {
		t.Fatal(err)
	}
	removeUploadedFile(root, "/uploads/avatars/ok.png")
	if _, err := os.Stat(safe); !os.IsNotExist(err) {
		t.Fatalf("expected avatar removed, err=%v", err)
	}

	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	var viewer models.User
	if err := app.accounts(models.UserKindAdmin).Where("username = ?", seed.ViewerUsername).First(&viewer).Error; err != nil {
		t.Fatal(err)
	}
	denied := doJSON(t, app, http.MethodPut, "/api/v1/users/"+formatUint(viewer.ID), admin, map[string]any{
		"avatar": "/uploads/avatars/../../secret.txt",
	})
	if denied.Code != http.StatusBadRequest {
		t.Fatalf("put traversal: %d %s", denied.Code, denied.Body.String())
	}
}

func TestWebUserListHonorsDataScope(t *testing.T) {
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
	var detailID uint
	for _, p := range decodePage[permissionDTO](t, doJSON(t, app, http.MethodGet, "/api/v1/permissions?pageSize=200", admin, nil)).Items {
		if p.Code == "user:detail" {
			detailID = p.ID
			break
		}
	}
	if detailID == 0 {
		t.Fatal("missing user:detail")
	}
	ids := append(append([]uint{}, opRole.PermissionIDs...), detailID)
	if w := doJSON(t, app, http.MethodPut, "/api/v1/roles/"+formatUint(opRole.ID)+"/permissions", admin, map[string]any{
		"permissionIds": ids,
	}); w.Code != http.StatusOK {
		t.Fatalf("grant perms: %d %s", w.Code, w.Body.String())
	}

	var ops models.Department
	if err := app.DB.Where("code = ?", "ops").First(&ops).Error; err != nil {
		t.Fatal(err)
	}
	scoped := models.User{
		Username: "opsweb", PasswordHash: "x", Nickname: "Ops Web",
		Status: "active", Department: "ops", DepartmentID: &ops.ID, Timezone: "Asia/Shanghai",
	}
	if err := app.accounts(models.UserKindWeb).Create(&scoped).Error; err != nil {
		t.Fatal(err)
	}
	var member models.User
	if err := app.accounts(models.UserKindWeb).Where("username = ?", seed.MemberUsername).First(&member).Error; err != nil {
		t.Fatal(err)
	}

	adminWeb := decodeUserPage(t, doJSON(t, app, http.MethodGet, "/api/v1/users?kind=web&pageSize=50", admin, nil))
	if len(adminWeb) < 2 {
		t.Fatalf("admin should see all web users, got %d", len(adminWeb))
	}

	opTok := loginOK(t, app, seed.OperatorUsername, seed.OperatorPassword)
	opWeb := decodeUserPage(t, doJSON(t, app, http.MethodGet, "/api/v1/users?kind=web&pageSize=50", opTok, nil))
	seen := map[string]bool{}
	for _, u := range opWeb {
		seen[u.Username] = true
		if u.Department != "" && u.Department != "ops" {
			t.Fatalf("operator saw out-of-scope web user %s dept=%s", u.Username, u.Department)
		}
	}
	if seen[seed.MemberUsername] {
		t.Fatal("operator should not see unscoped seed web user")
	}
	if !seen["opsweb"] {
		t.Fatal("operator should see web user in ops department")
	}

	hidden := doJSON(t, app, http.MethodGet, "/api/v1/users/"+formatUint(member.ID)+"?kind=web", opTok, nil)
	if hidden.Code != http.StatusNotFound {
		t.Fatalf("cross-dept web get want 404 got %d %s", hidden.Code, hidden.Body.String())
	}
	visible := doJSON(t, app, http.MethodGet, "/api/v1/users/"+formatUint(scoped.ID)+"?kind=web", opTok, nil)
	if visible.Code != http.StatusOK {
		t.Fatalf("in-scope web get: %d %s", visible.Code, visible.Body.String())
	}

	exported := doJSON(t, app, http.MethodGet, "/api/v1/users/export?kind=web", opTok, nil)
	if exported.Code != http.StatusOK {
		t.Fatalf("export: %d %s", exported.Code, exported.Body.String())
	}
	csv := exported.Body.String()
	if strings.Contains(csv, seed.MemberUsername) {
		t.Fatal("operator export should omit unscoped web user")
	}
	if !strings.Contains(csv, "opsweb") {
		t.Fatal("operator export should include in-scope web user")
	}
}

func doJSONOrigin(t *testing.T, app *App, path string, payload any, origin string) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(string(raw)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", origin)
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	return w
}
