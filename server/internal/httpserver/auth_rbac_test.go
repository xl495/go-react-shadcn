package httpserver

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go-react-shadcn/internal/config"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/seed"
	"go-react-shadcn/internal/store"
)

type envelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func testApp(t *testing.T) *App {
	t.Helper()
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	cfg := config.Config{
		Port:         "0",
		DatabasePath: filepath.Join(dir, "test.db"),
		JWTSecret:    "test-secret",
		JWTTTL:       time.Hour,
		CaptchaDebug: false,
		CORSOrigin:   "http://localhost:5173",
		UploadDir:    filepath.Join(dir, "uploads"),
	}
	app, err := New(cfg, db)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	t.Cleanup(app.Close)
	return app
}

func doJSON(t *testing.T, app *App, method, path, token string, payload any) *httptest.ResponseRecorder {
	t.Helper()
	var body *bytes.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		body = bytes.NewReader(raw)
	} else {
		body = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, body)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	return w
}

func decodeEnv(t *testing.T, w *httptest.ResponseRecorder) envelope {
	t.Helper()
	var env envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope from %s: %v", w.Body.String(), err)
	}
	return env
}

type pageEnvelope[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
}

func decodePage[T any](t *testing.T, w *httptest.ResponseRecorder) pageEnvelope[T] {
	t.Helper()
	var page pageEnvelope[T]
	if err := json.Unmarshal(decodeEnv(t, w).Data, &page); err != nil {
		t.Fatalf("decode page: %v body=%s", err, w.Body.String())
	}
	return page
}

func decodeUserPage(t *testing.T, w *httptest.ResponseRecorder) []userDTO {
	t.Helper()
	return decodePage[userDTO](t, w).Items
}

func decodeOpLogPage(t *testing.T, w *httptest.ResponseRecorder) []models.OpLog {
	t.Helper()
	return decodePage[models.OpLog](t, w).Items
}

func decodeLoginLogPage(t *testing.T, w *httptest.ResponseRecorder) []models.LoginLog {
	t.Helper()
	return decodePage[models.LoginLog](t, w).Items
}

func decodeRolePage(t *testing.T, w *httptest.ResponseRecorder) []roleDTO {
	t.Helper()
	return decodePage[roleDTO](t, w).Items
}

func issueCaptcha(t *testing.T, app *App) (id, answer, image string) {
	t.Helper()
	w := doJSON(t, app, http.MethodGet, "/api/v1/auth/captcha", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("captcha status=%d body=%s", w.Code, w.Body.String())
	}
	env := decodeEnv(t, w)
	var ch struct {
		CaptchaID string `json:"captchaId"`
		Image     string `json:"image"`
	}
	if err := json.Unmarshal(env.Data, &ch); err != nil {
		t.Fatalf("captcha data: %v", err)
	}
	if ch.CaptchaID == "" {
		t.Fatal("expected captcha id")
	}
	if !strings.Contains(ch.Image, "base64") && !strings.HasPrefix(ch.Image, "data:image") {
		t.Fatalf("expected image payload, got %q", truncate(ch.Image, 80))
	}
	answer = app.Captcha.Peek(ch.CaptchaID)
	if answer == "" {
		t.Fatal("store has no captcha answer")
	}
	return ch.CaptchaID, answer, ch.Image
}

func login(t *testing.T, app *App, username, password, captchaID, captchaCode string) *httptest.ResponseRecorder {
	t.Helper()
	return doJSON(t, app, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"username":    username,
		"password":    password,
		"captchaId":   captchaID,
		"captchaCode": captchaCode,
	})
}

func loginOK(t *testing.T, app *App, username, password string) string {
	t.Helper()
	id, answer, _ := issueCaptcha(t, app)
	w := login(t, app, username, password, id, answer)
	if w.Code != http.StatusOK {
		t.Fatalf("login %s status=%d body=%s", username, w.Code, w.Body.String())
	}
	env := decodeEnv(t, w)
	var data struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		t.Fatalf("login data: %v", err)
	}
	if data.Token == "" || strings.Count(data.Token, ".") != 2 {
		t.Fatalf("expected jwt, got %q", data.Token)
	}
	return data.Token
}

func TestHealthEndpoint(t *testing.T) {
	app := testApp(t)
	w := doJSON(t, app, http.MethodGet, "/health", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("health status=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"ok"`) {
		t.Fatalf("health body missing ok: %s", w.Body.String())
	}
	env := decodeEnv(t, w)
	if env.Message != "ok" {
		t.Fatalf("message=%q", env.Message)
	}
	var data struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		t.Fatalf("health data: %v", err)
	}
	if data.Status != "ok" {
		t.Fatalf("status=%q", data.Status)
	}

	api := doJSON(t, app, http.MethodGet, "/api/v1/health", "", nil)
	if api.Code != http.StatusOK || !strings.Contains(api.Body.String(), `"ok"`) {
		t.Fatalf("api health: %d %s", api.Code, api.Body.String())
	}
}

func TestOperationLogsRecordLoginAndMutations(t *testing.T) {
	app := testApp(t)
	token := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)

	created := doJSON(t, app, http.MethodPost, "/api/v1/configs", token, map[string]string{
		"key": "app.audit_probe", "value": "1", "name": "audit probe", "group": "app",
	})
	if created.Code != http.StatusOK {
		t.Fatalf("create config: %d %s", created.Code, created.Body.String())
	}

	listed := doJSON(t, app, http.MethodGet, "/api/v1/logs/login?username=adm&status=success", token, nil)
	if listed.Code != http.StatusOK {
		t.Fatalf("list login logs: %d %s", listed.Code, listed.Body.String())
	}
	loginLogs := decodeLoginLogPage(t, listed)
	if len(loginLogs) == 0 {
		t.Fatal("expected login audit row")
	}
	foundLogin := false
	for _, row := range loginLogs {
		if row.Username == seed.AdminUsername && row.Status == "success" {
			foundLogin = true
			break
		}
	}
	if !foundLogin {
		t.Fatalf("login log missing: %+v", loginLogs)
	}

	cfgLogs := doJSON(t, app, http.MethodGet, "/api/v1/logs?module=config", token, nil)
	if cfgLogs.Code != http.StatusOK {
		t.Fatalf("list config logs: %d %s", cfgLogs.Code, cfgLogs.Body.String())
	}
	rows := decodeOpLogPage(t, cfgLogs)
	foundWrite := false
	for _, row := range rows {
		if row.Username == seed.AdminUsername && row.Action == "create" && row.Path == "/api/v1/configs" && row.Status == http.StatusOK {
			foundWrite = true
			break
		}
	}
	if !foundWrite {
		t.Fatalf("config mutation log missing: %+v", rows)
	}
	if n := app.countLogs(seed.AdminUsername, "config", "create"); n < 1 {
		t.Fatalf("db config logs=%d", n)
	}
}

func TestCaptchaIssued(t *testing.T) {
	app := testApp(t)
	id, answer, image := issueCaptcha(t, app)
	if id == "" || answer == "" || image == "" {
		t.Fatalf("incomplete challenge id=%q answer=%q imageLen=%d", id, answer, len(image))
	}
}

func TestLoginRejectsMissingAndWrongCaptcha(t *testing.T) {
	app := testApp(t)
	id, _, _ := issueCaptcha(t, app)

	missing := login(t, app, seed.AdminUsername, seed.AdminPassword, "", "")
	if missing.Code == http.StatusOK {
		t.Fatal("missing captcha must not yield JWT")
	}
	if tokenFrom(t, missing) != "" {
		t.Fatal("missing captcha leaked a token")
	}

	wrong := login(t, app, seed.AdminUsername, seed.AdminPassword, id, "0000")
	if wrong.Code == http.StatusOK {
		t.Fatal("wrong captcha must not yield JWT")
	}
	if tokenFrom(t, wrong) != "" {
		t.Fatal("wrong captcha leaked a token")
	}

	// consumed by the wrong attempt; a later correct guess on the same id must fail
	again := login(t, app, seed.AdminUsername, seed.AdminPassword, id, app.Captcha.Peek(id))
	if again.Code == http.StatusOK {
		t.Fatal("consumed captcha must not be reusable")
	}
}

func TestLoginWrongPasswordNoJWT(t *testing.T) {
	app := testApp(t)
	id, answer, _ := issueCaptcha(t, app)
	w := login(t, app, seed.AdminUsername, "not-the-password", id, answer)
	if w.Code == http.StatusOK || tokenFrom(t, w) != "" {
		t.Fatalf("wrong password issued token: %s", w.Body.String())
	}
}

func TestAdminLoginJWTAndAdminAPI(t *testing.T) {
	app := testApp(t)
	token := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	w := doJSON(t, app, http.MethodGet, "/api/v1/users", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("admin users status=%d body=%s", w.Code, w.Body.String())
	}
	users := decodeUserPage(t, w)
	if len(users) < 2 {
		t.Fatalf("expected seeded users, got %d", len(users))
	}
}

func TestGetUserDetailAllowDeny(t *testing.T) {
	app := testApp(t)
	adminTok := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	limitedTok := loginOK(t, app, seed.OperatorUsername, seed.OperatorPassword)

	listed := doJSON(t, app, http.MethodGet, "/api/v1/users", adminTok, nil)
	if listed.Code != http.StatusOK {
		t.Fatalf("list users: %d %s", listed.Code, listed.Body.String())
	}
	users := decodeUserPage(t, listed)
	var admin userDTO
	for _, u := range users {
		if u.Username == seed.AdminUsername {
			admin = u
		}
	}
	if admin.ID == 0 {
		t.Fatal("seeded admin missing from list")
	}

	got := doJSON(t, app, http.MethodGet, "/api/v1/users/"+itoa(admin.ID), adminTok, nil)
	if got.Code != http.StatusOK {
		t.Fatalf("admin get user: %d %s", got.Code, got.Body.String())
	}
	var detail userDTO
	if err := json.Unmarshal(decodeEnv(t, got).Data, &detail); err != nil {
		t.Fatal(err)
	}
	if detail.Username != seed.AdminUsername {
		t.Fatalf("username=%q", detail.Username)
	}
	if detail.Phone == "" || detail.Avatar == "" {
		t.Fatalf("expected phone and avatar, got phone=%q avatar=%q", detail.Phone, detail.Avatar)
	}

	denied := doJSON(t, app, http.MethodGet, "/api/v1/users/"+itoa(admin.ID), limitedTok, nil)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("operator get user should 403, got %d %s", denied.Code, denied.Body.String())
	}

	if w := doJSON(t, app, http.MethodGet, "/api/v1/dicts", adminTok, nil); w.Code != http.StatusOK {
		t.Fatalf("admin dicts: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/configs", adminTok, nil); w.Code != http.StatusOK {
		t.Fatalf("admin configs: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/logs", adminTok, nil); w.Code != http.StatusOK {
		t.Fatalf("admin logs: %d %s", w.Code, w.Body.String())
	}
}

func TestViewerDeniedAdminAPI(t *testing.T) {
	app := testApp(t)
	token := loginOK(t, app, seed.ViewerUsername, seed.ViewerPassword)
	w := doJSON(t, app, http.MethodGet, "/api/v1/users", token, nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("viewer users status=%d body=%s", w.Code, w.Body.String())
	}
	if tokenFrom(t, w) != "" {
		t.Fatal("denied response should not include a token")
	}
}

func TestAssigningPermissionChangesCasbinEnforce(t *testing.T) {
	app := testApp(t)
	adminTok := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)

	before, err := app.Enforcer.Enforce(seed.ViewerUsername, "/api/v1/users", "GET")
	if err != nil {
		t.Fatalf("enforce before: %v", err)
	}
	if before {
		t.Fatal("viewer should not have users GET before assignment")
	}

	denied := doJSON(t, app, http.MethodGet, "/api/v1/users", loginOK(t, app, seed.ViewerUsername, seed.ViewerPassword), nil)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("viewer still expected 403, got %d %s", denied.Code, denied.Body.String())
	}

	created := doJSON(t, app, http.MethodPost, "/api/v1/permissions", adminTok, map[string]string{
		"name":   "用户只读",
		"code":   "user:list-copy",
		"path":   "/api/v1/users",
		"method": "GET",
	})
	if created.Code != http.StatusOK {
		t.Fatalf("create permission status=%d body=%s", created.Code, created.Body.String())
	}
	var permEnv envelope
	if err := json.Unmarshal(created.Body.Bytes(), &permEnv); err != nil {
		t.Fatal(err)
	}
	var perm permissionDTO
	if err := json.Unmarshal(permEnv.Data, &perm); err != nil {
		t.Fatal(err)
	}

	rolesResp := doJSON(t, app, http.MethodGet, "/api/v1/roles", adminTok, nil)
	if rolesResp.Code != http.StatusOK {
		t.Fatalf("roles status=%d body=%s", rolesResp.Code, rolesResp.Body.String())
	}
	roles := decodeRolePage(t, rolesResp)
	var viewer roleDTO
	var meID, dashID uint
	for _, r := range roles {
		if r.Code == seed.RoleViewer {
			viewer = r
			for _, p := range r.Permissions {
				switch p.Code {
				case "me:read":
					meID = p.ID
				case "dashboard:read":
					dashID = p.ID
				}
			}
		}
	}
	if viewer.ID == 0 || meID == 0 || dashID == 0 {
		t.Fatalf("missing viewer seed perms: %+v", viewer)
	}

	assign := doJSON(t, app, http.MethodPut, "/api/v1/roles/"+itoa(viewer.ID)+"/permissions", adminTok, map[string]any{
		"permissionIds": []uint{meID, dashID, perm.ID},
	})
	if assign.Code != http.StatusOK {
		t.Fatalf("assign status=%d body=%s", assign.Code, assign.Body.String())
	}

	after, err := app.Enforcer.Enforce(seed.ViewerUsername, "/api/v1/users", "GET")
	if err != nil {
		t.Fatalf("enforce after: %v", err)
	}
	if !after {
		t.Fatal("viewer should pass Casbin after permission assignment")
	}

	allowed := doJSON(t, app, http.MethodGet, "/api/v1/users", loginOK(t, app, seed.ViewerUsername, seed.ViewerPassword), nil)
	if allowed.Code != http.StatusOK {
		t.Fatalf("viewer users after grant status=%d body=%s", allowed.Code, allowed.Body.String())
	}
}

func TestOperatorButtonPermissions(t *testing.T) {
	app := testApp(t)
	token := loginOK(t, app, seed.OperatorUsername, seed.OperatorPassword)

	me := doJSON(t, app, http.MethodGet, "/api/v1/auth/me", token, nil)
	if me.Code != http.StatusOK {
		t.Fatalf("me status=%d body=%s", me.Code, me.Body.String())
	}
	env := decodeEnv(t, me)
	var profile userDTO
	if err := json.Unmarshal(env.Data, &profile); err != nil {
		t.Fatal(err)
	}
	has := map[string]bool{}
	for _, c := range profile.PermissionCodes {
		has[c] = true
	}
	if !has["user:list"] || !has["user:create"] {
		t.Fatalf("operator should have list+create buttons, got %v", profile.PermissionCodes)
	}
	if has["user:delete"] || has["user:roles"] || has["role:create"] || has["*"] {
		t.Fatalf("operator should not have delete/assign/create-role, got %v", profile.PermissionCodes)
	}

	if w := doJSON(t, app, http.MethodGet, "/api/v1/users", token, nil); w.Code != http.StatusOK {
		t.Fatalf("operator list users: %d %s", w.Code, w.Body.String())
	}
	created := doJSON(t, app, http.MethodPost, "/api/v1/users", token, map[string]any{
		"username": "tmp-op",
		"password": "tmp-op-pass",
		"roleIds":  []uint{},
	})
	if created.Code != http.StatusOK {
		t.Fatalf("operator create user (button): %d %s", created.Code, created.Body.String())
	}
	var createdEnv envelope
	if err := json.Unmarshal(created.Body.Bytes(), &createdEnv); err != nil {
		t.Fatal(err)
	}
	var createdUser userDTO
	if err := json.Unmarshal(createdEnv.Data, &createdUser); err != nil {
		t.Fatal(err)
	}

	del := doJSON(t, app, http.MethodDelete, "/api/v1/users/"+itoa(createdUser.ID), token, nil)
	if del.Code != http.StatusForbidden {
		t.Fatalf("operator delete user should be 403, got %d %s", del.Code, del.Body.String())
	}
	assign := doJSON(t, app, http.MethodPut, "/api/v1/users/"+itoa(createdUser.ID)+"/roles", token, map[string]any{
		"roleIds": []uint{},
	})
	if assign.Code != http.StatusForbidden {
		t.Fatalf("operator assign roles should be 403, got %d %s", assign.Code, assign.Body.String())
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/roles", token, map[string]string{
		"name": "x", "code": "x",
	}); w.Code != http.StatusForbidden {
		t.Fatalf("operator create role should be 403, got %d %s", w.Code, w.Body.String())
	}
}

func TestAvatarUploadAndPhone(t *testing.T) {
	app := testApp(t)
	token := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	me := doJSON(t, app, http.MethodGet, "/api/v1/auth/me", token, nil)
	env := decodeEnv(t, me)
	var profile userDTO
	if err := json.Unmarshal(env.Data, &profile); err != nil {
		t.Fatal(err)
	}
	if profile.Phone == "" {
		t.Fatal("seeded user missing phone")
	}
	if profile.Avatar == "" {
		t.Fatal("seeded user missing avatar")
	}
	seeded := doJSON(t, app, http.MethodGet, profile.Avatar, "", nil)
	if seeded.Code != http.StatusOK || seeded.Body.Len() == 0 {
		t.Fatalf("seeded avatar not served: %d", seeded.Code)
	}

	var buf bytes.Buffer
	mw := newMultipartPNG(&buf)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/avatar", &buf)
	req.Header.Set("Content-Type", mw)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.Router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("upload avatar: %d %s", rec.Code, rec.Body.String())
	}
	upEnv := decodeEnv(t, rec)
	var after userDTO
	if err := json.Unmarshal(upEnv.Data, &after); err != nil {
		t.Fatal(err)
	}
	if after.Avatar == "" || after.Avatar == profile.Avatar {
		t.Fatalf("avatar path not updated: %q -> %q", profile.Avatar, after.Avatar)
	}
	served := doJSON(t, app, http.MethodGet, after.Avatar, "", nil)
	if served.Code != http.StatusOK || served.Body.Len() == 0 {
		t.Fatalf("uploaded avatar not served: %d", served.Code)
	}
}

func newMultipartPNG(buf *bytes.Buffer) string {
	w := multipart.NewWriter(buf)
	part, _ := w.CreateFormFile("file", "dot.png")
	_, _ = part.Write(tinyPNG())
	_ = w.Close()
	return w.FormDataContentType()
}

func tinyPNG() []byte {
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, 0x89, 0x00, 0x00, 0x00,
		0x0a, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00, 0x00, 0x00, 0x49,
		0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
	}
}

func TestProfileAndPassword(t *testing.T) {
	app := testApp(t)
	token := loginOK(t, app, seed.ViewerUsername, seed.ViewerPassword)

	me := doJSON(t, app, http.MethodGet, "/api/v1/auth/me", token, nil)
	if me.Code != http.StatusOK {
		t.Fatalf("me: %s", me.Body.String())
	}
	env := decodeEnv(t, me)
	var profile userDTO
	if err := json.Unmarshal(env.Data, &profile); err != nil {
		t.Fatal(err)
	}
	if profile.Nickname == "" || profile.Email == "" || profile.Phone == "" || profile.Department == "" {
		t.Fatalf("seeded profile incomplete: %+v", profile)
	}
	if profile.LastLoginAt == nil {
		t.Fatal("login should stamp lastLoginAt")
	}

	upd := doJSON(t, app, http.MethodPut, "/api/v1/auth/profile", token, map[string]string{
		"nickname":   "访客改名",
		"email":      "viewer2@latch.local",
		"phone":      "13900000000",
		"gender":     "female",
		"department": "market",
		"title":      "观察员",
		"remark":     "updated",
	})
	if upd.Code != http.StatusOK {
		t.Fatalf("profile update: %d %s", upd.Code, upd.Body.String())
	}
	updEnv := decodeEnv(t, upd)
	var after userDTO
	if err := json.Unmarshal(updEnv.Data, &after); err != nil {
		t.Fatal(err)
	}
	if after.Nickname != "访客改名" || after.Email != "viewer2@latch.local" || after.Department != "market" {
		t.Fatalf("profile not saved: %+v", after)
	}

	bad := doJSON(t, app, http.MethodPut, "/api/v1/auth/password", token, map[string]string{
		"oldPassword": "wrong",
		"newPassword": "new-pass-1",
	})
	if bad.Code == http.StatusOK {
		t.Fatal("wrong current password must fail")
	}
	okPwd := doJSON(t, app, http.MethodPut, "/api/v1/auth/password", token, map[string]string{
		"oldPassword": seed.ViewerPassword,
		"newPassword": "new-pass-1",
	})
	if okPwd.Code != http.StatusOK {
		t.Fatalf("password change: %d %s", okPwd.Code, okPwd.Body.String())
	}
	oldID, oldAns, _ := issueCaptcha(t, app)
	if login(t, app, seed.ViewerUsername, seed.ViewerPassword, oldID, oldAns).Code == http.StatusOK {
		t.Fatal("old password should no longer work")
	}
	id, ans, _ := issueCaptcha(t, app)
	if w := login(t, app, seed.ViewerUsername, "new-pass-1", id, ans); w.Code != http.StatusOK {
		t.Fatalf("new password login: %d %s", w.Code, w.Body.String())
	}
}

func TestUserFieldsBoundToDict(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)

	gender := lookupDictValues(t, app, admin, seed.DictGender)
	status := lookupDictValues(t, app, admin, seed.DictUserStatus)
	dept := lookupDictValues(t, app, admin, seed.DictDepartment)
	if len(gender) < 2 || len(status) < 2 || len(dept) < 3 {
		t.Fatalf("seeded user dicts incomplete gender=%v status=%v dept=%v", gender, status, dept)
	}

	list := doJSON(t, app, http.MethodGet, "/api/v1/users", admin, nil)
	if list.Code != http.StatusOK {
		t.Fatalf("users: %d %s", list.Code, list.Body.String())
	}
	users := decodeUserPage(t, list)
	if len(users) == 0 {
		t.Fatal("expected seeded users")
	}
	for _, u := range users {
		if u.Gender != "" && !gender[u.Gender] {
			t.Fatalf("user %s gender %q not in %s", u.Username, u.Gender, seed.DictGender)
		}
		if u.Status == "" || !status[u.Status] {
			t.Fatalf("user %s status %q not in %s", u.Username, u.Status, seed.DictUserStatus)
		}
		if u.Department != "" && !dept[u.Department] {
			t.Fatalf("user %s department %q not in %s", u.Username, u.Department, seed.DictDepartment)
		}
	}

	detail := doJSON(t, app, http.MethodGet, "/api/v1/users/"+itoa(users[0].ID), admin, nil)
	if detail.Code != http.StatusOK {
		t.Fatalf("user detail: %d %s", detail.Code, detail.Body.String())
	}
	var one userDTO
	if err := json.Unmarshal(decodeEnv(t, detail).Data, &one); err != nil {
		t.Fatal(err)
	}
	if one.Gender != "" && !gender[one.Gender] {
		t.Fatalf("detail gender %q not in dict", one.Gender)
	}
	if one.Department != "" && !dept[one.Department] {
		t.Fatalf("detail department %q not in dict", one.Department)
	}

	bad := doJSON(t, app, http.MethodPut, "/api/v1/auth/profile", admin, map[string]string{
		"nickname": "x", "gender": "not-a-gender", "department": "tech",
	})
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("invalid gender should 400, got %d %s", bad.Code, bad.Body.String())
	}

	okUpd := doJSON(t, app, http.MethodPut, "/api/v1/auth/profile", admin, map[string]string{
		"nickname": "系统管理员", "email": "admin@latch.local", "phone": "13800000001",
		"gender": "male", "department": "ops", "title": "负责人",
	})
	if okUpd.Code != http.StatusOK {
		t.Fatalf("valid dict profile: %d %s", okUpd.Code, okUpd.Body.String())
	}

	badCreate := doJSON(t, app, http.MethodPost, "/api/v1/users", admin, map[string]any{
		"username": "dict-user", "password": "pass-1234", "status": "nope",
	})
	if badCreate.Code != http.StatusBadRequest {
		t.Fatalf("invalid status should 400, got %d %s", badCreate.Code, badCreate.Body.String())
	}
}

func lookupDictValues(t *testing.T, app *App, token, code string) map[string]bool {
	t.Helper()
	w := doJSON(t, app, http.MethodGet, "/api/v1/dicts/by/"+code, token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("lookup %s: %d %s", code, w.Code, w.Body.String())
	}
	var pack struct {
		Items []struct {
			Value string `json:"value"`
		} `json:"items"`
	}
	if err := json.Unmarshal(decodeEnv(t, w).Data, &pack); err != nil {
		t.Fatal(err)
	}
	out := map[string]bool{}
	for _, it := range pack.Items {
		out[it.Value] = true
	}
	return out
}

func TestFoundationDictConfigLogs(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	viewer := loginOK(t, app, seed.ViewerUsername, seed.ViewerPassword)
	operator := loginOK(t, app, seed.OperatorUsername, seed.OperatorPassword)

	if w := doJSON(t, app, http.MethodGet, "/api/v1/dicts", viewer, nil); w.Code != http.StatusForbidden {
		t.Fatalf("viewer dicts should 403, got %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/dicts", operator, nil); w.Code != http.StatusOK {
		t.Fatalf("operator dicts: %d %s", w.Code, w.Body.String())
	}

	lookup := doJSON(t, app, http.MethodGet, "/api/v1/dicts/by/sys_gender", viewer, nil)
	if lookup.Code != http.StatusOK {
		t.Fatalf("seeded gender lookup: %d %s", lookup.Code, lookup.Body.String())
	}
	lookEnv := decodeEnv(t, lookup)
	var pack struct {
		Code  string `json:"code"`
		Items []struct {
			Value string `json:"value"`
			Label string `json:"label"`
		} `json:"items"`
	}
	if err := json.Unmarshal(lookEnv.Data, &pack); err != nil {
		t.Fatal(err)
	}
	if pack.Code != "sys_gender" || len(pack.Items) < 2 {
		t.Fatalf("gender dict incomplete: %+v", pack)
	}

	created := doJSON(t, app, http.MethodPost, "/api/v1/dicts", admin, map[string]string{
		"code": "biz_level", "name": "优先级",
	})
	if created.Code != http.StatusOK {
		t.Fatalf("create dict: %d %s", created.Code, created.Body.String())
	}
	var typ models.DictType
	if err := json.Unmarshal(decodeEnv(t, created).Data, &typ); err != nil {
		t.Fatal(err)
	}
	item := doJSON(t, app, http.MethodPost, "/api/v1/dicts/"+itoa(typ.ID)+"/items", admin, map[string]any{
		"label": "高", "value": "high", "sort": 1,
	})
	if item.Code != http.StatusOK {
		t.Fatalf("create item: %d %s", item.Code, item.Body.String())
	}
	got := doJSON(t, app, http.MethodGet, "/api/v1/dicts/by/biz_level", admin, nil)
	if got.Code != http.StatusOK {
		t.Fatalf("lookup new dict: %d %s", got.Code, got.Body.String())
	}
	if !strings.Contains(got.Body.String(), `"high"`) {
		t.Fatalf("new dict item missing: %s", got.Body.String())
	}

	if w := doJSON(t, app, http.MethodPost, "/api/v1/dicts", operator, map[string]string{
		"code": "x", "name": "x",
	}); w.Code != http.StatusForbidden {
		t.Fatalf("operator create dict should 403, got %d", w.Code)
	}

	cfg := doJSON(t, app, http.MethodPost, "/api/v1/configs", admin, map[string]string{
		"key": "app.test_flag", "value": "on", "name": "测试开关", "group": "app",
	})
	if cfg.Code != http.StatusOK {
		t.Fatalf("create config: %d %s", cfg.Code, cfg.Body.String())
	}
	var saved models.SysConfig
	if err := json.Unmarshal(decodeEnv(t, cfg).Data, &saved); err != nil {
		t.Fatal(err)
	}
	upd := doJSON(t, app, http.MethodPut, "/api/v1/configs/"+itoa(saved.ID), admin, map[string]string{
		"value": "off", "name": "测试开关",
	})
	if upd.Code != http.StatusOK {
		t.Fatalf("update config: %d %s", upd.Code, upd.Body.String())
	}
	listed := doJSON(t, app, http.MethodGet, "/api/v1/configs", admin, nil)
	if listed.Code != http.StatusOK || !strings.Contains(listed.Body.String(), `"off"`) {
		t.Fatalf("config list missing update: %s", listed.Body.String())
	}

	loginLogs := doJSON(t, app, http.MethodGet, "/api/v1/logs/login?username="+seed.AdminUsername, admin, nil)
	if loginLogs.Code != http.StatusOK {
		t.Fatalf("login logs: %d %s", loginLogs.Code, loginLogs.Body.String())
	}
	if len(decodeLoginLogPage(t, loginLogs)) == 0 {
		t.Fatalf("expected login log after loginOK")
	}

	logs := doJSON(t, app, http.MethodGet, "/api/v1/logs", admin, nil)
	if logs.Code != http.StatusOK {
		t.Fatalf("logs: %d %s", logs.Code, logs.Body.String())
	}
	if !strings.Contains(logs.Body.String(), "biz_level") && !strings.Contains(logs.Body.String(), "/api/v1/dicts") {
		t.Fatalf("expected dict mutation log: %s", logs.Body.String())
	}
}

func tokenFrom(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var env envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		return ""
	}
	var data struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(env.Data, &data)
	return data.Token
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func itoa(n uint) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
