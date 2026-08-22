package httpserver

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/seed"
	"go-react-shadcn/internal/totp"
)

func TestCoverageBoostRemainingHandlers(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	operator := loginOK(t, app, seed.OperatorUsername, seed.OperatorPassword)

	coverHelpers(t, app)
	coverGzipString(t, app)
	coverMethodNotAllowed(t, app)
	coverNavBranches(t, app, admin)
	coverMailAndLogs(t, app, admin)
	coverDepartmentsAndUsers(t, app, admin)
	coverRolesPermsAnnounce(t, app, admin)
	coverImportErrors(t, app)
	coverTotpLogin(t, app, operator)
	coverAvatarForUser(t, app, admin)

	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/google/unbind", admin, nil); w.Code != http.StatusOK {
		t.Fatalf("unbind empty: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/google/bind", admin, map[string]string{"idToken": "x"}); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("bind disabled: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodPut, "/api/v1/users/batch-status", admin, map[string]any{"ids": []uint{1}, "status": "nope"}); w.Code != http.StatusBadRequest {
		t.Fatalf("batch bad: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/online-sessions", admin, nil); w.Code != http.StatusOK {
		t.Fatalf("online: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/auth/sessions", admin, nil); w.Code != http.StatusOK {
		t.Fatalf("own sessions: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/auth/login-logs", admin, nil); w.Code != http.StatusOK {
		t.Fatalf("own logs: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/auth/menus", admin, nil); w.Code != http.StatusOK {
		t.Fatalf("menus: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/auth/web-menus", admin, nil); w.Code != http.StatusOK {
		t.Fatalf("web menus: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/dicts/by/missing_dict", admin, nil); w.Code != http.StatusNotFound {
		t.Fatalf("missing dict: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/mail/test", admin, map[string]string{"to": "not-an-email"}); w.Code != http.StatusBadRequest {
		t.Fatalf("test mail: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/users/1/unlock", admin, nil); w.Code != http.StatusOK {
		t.Fatalf("unlock: %d %s", w.Code, w.Body.String())
	}
}

func TestCoverageBoostHealthWhenDBClosed(t *testing.T) {
	app := testApp(t)
	sqlDB, err := app.DB.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}
	if w := doJSON(t, app, http.MethodGet, "/health", "", nil); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("health down: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodGet, "/ready", "", nil); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("ready down: %d %s", w.Code, w.Body.String())
	}
}

func coverHelpers(t *testing.T, app *App) {
	t.Helper()
	if clipLen("abcdef", 3) != "abc" || clipLen("ab", 3) != "ab" {
		t.Fatal("clipLen")
	}
	if clip("abcdef", 3) != "" || clip("ab", 3) != "ab" {
		t.Fatal("clip")
	}
	if googleUsername("Hello.World@ex.com", "") == "" {
		t.Fatal("googleUsername email")
	}
	if googleUsername("!!!@@@", "sub-123") == "" {
		t.Fatal("googleUsername sub")
	}
	if googleUsername("", "") != "google" {
		t.Fatal("googleUsername empty")
	}
	if len(googleUsername(strings.Repeat("a", 80)+"@x.com", "s")) > 48 {
		t.Fatal("googleUsername long")
	}
	name, err := app.uniqueUsername(models.UserKindAdmin, "admin")
	if err != nil || name == "admin" {
		t.Fatalf("uniqueUsername: %q %v", name, err)
	}
	_, _ = app.uniqueUsername(models.UserKindAdmin, strings.Repeat("u", 70))
	if !isUniqueViolation(errors.New("UNIQUE constraint failed: users.username")) {
		t.Fatal("isUniqueViolation")
	}
	if isUniqueViolation(nil) || isUniqueViolation(errors.New("other")) {
		t.Fatal("isUniqueViolation false")
	}
	app.HTTP = nil
	if app.httpClient() == nil {
		t.Fatal("httpClient")
	}
	if errRoleNotFound.Error() != "role not found" {
		t.Fatal("errString")
	}
	lines := appendError(nil, 1, string([]byte{0xff}))
	for i := 0; i < 35; i++ {
		lines = appendError(lines, i, "x")
	}
	if len(lines) > 30 {
		t.Fatalf("appendError cap %d", len(lines))
	}
	if !isAnomalousLogin(models.User{LastLoginIP: "1.2.3.4"}, "8.8.8.8") {
		t.Fatal("anomalous")
	}
	if isAnomalousLogin(models.User{LastLoginIP: "1.2.3.4"}, "1.2.9.9") {
		t.Fatal("same /16")
	}
	if isAnomalousLogin(models.User{}, "1.1.1.1") || isAnomalousLogin(models.User{LastLoginIP: "1.1.1.1"}, "1.1.1.1") {
		t.Fatal("not anomalous")
	}
	if isAnomalousLogin(models.User{LastLoginIP: "fe80::1"}, "fe80::2") == false {
		t.Fatal("non-ipv4")
	}

	var hq models.Department
	if err := app.DB.Where("code = ?", "hq").First(&hq).Error; err != nil {
		t.Fatal(err)
	}
	child := models.Department{Name: "覆盖子部门", Code: "hq-cov", ParentID: &hq.ID, Status: "active"}
	if err := app.DB.Create(&child).Error; err != nil {
		t.Fatal(err)
	}
	app.refreshDepartments()
	ids := app.deptSubtreeIDs(hq.ID)
	if len(ids) < 2 {
		t.Fatalf("subtree %v", ids)
	}

	prev := app.Cfg.DevMode
	app.Cfg.DevMode = false
	_ = app.passwordIssue("short", "u")
	app.Cfg.DevMode = prev

	_ = app.recaptchaMinScore()
	setCfg(t, app, "auth.recaptcha_min_score", "bad")
	if app.recaptchaMinScore() != 0.5 {
		t.Fatal("score bad")
	}
	setCfg(t, app, "auth.recaptcha_min_score", "-1")
	if app.recaptchaMinScore() != 0 {
		t.Fatal("score low")
	}
	setCfg(t, app, "auth.recaptcha_min_score", "2")
	if app.recaptchaMinScore() != 1 {
		t.Fatal("score high")
	}
	setCfg(t, app, "auth.captcha_provider", "recaptcha")
	if app.captchaProvider() != "recaptcha" {
		t.Fatal("provider recaptcha")
	}
	setCfg(t, app, "auth.captcha_provider", "turnstile")
	if app.captchaProvider() != "turnstile" {
		t.Fatal("provider turnstile")
	}
	setCfg(t, app, "auth.captcha_provider", "none")
	if app.captchaProvider() != "none" {
		t.Fatal("provider none")
	}
	setCfg(t, app, "auth.captcha_provider", "image")
	_ = app.sysValue("missing.sys.key")
	if err := app.DB.Model(&models.SysConfig{}).Where(`"key" = ?`, "mail.password").Update("value", "enc:v1:not-valid").Error; err != nil {
		t.Fatal(err)
	}
	app.warnSealedConfigs()
	app.warnSealedConfigs()
	_ = app.consumeRecoveryCode(models.User{ID: 1, Kind: models.UserKindAdmin}, "")

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	writeCSV(c, "x.csv", []string{"a", "b"}, [][]string{{"1", "2"}})
	if !strings.Contains(rec.Body.String(), "1,2") {
		t.Fatalf("writeCSV %s", rec.Body.String())
	}
}

func coverGzipString(t *testing.T, app *App) {
	t.Helper()
	app.Router.GET("/__gzip_str", func(c *gin.Context) {
		c.String(http.StatusOK, "hello-gzip")
		c.Writer.Flush()
	})
	req := httptest.NewRequest(http.MethodGet, "/__gzip_str", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("gzip str encoding=%q", w.Header().Get("Content-Encoding"))
	}
	zr, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(zr)
	_ = zr.Close()
	if err != nil || string(body) != "hello-gzip" {
		t.Fatalf("gzip str body=%q err=%v", body, err)
	}
	head := httptest.NewRequest(http.MethodHead, "/live", nil)
	head.Header.Set("Accept-Encoding", "gzip")
	hw := httptest.NewRecorder()
	app.Router.ServeHTTP(hw, head)
}

func coverMethodNotAllowed(t *testing.T, app *App) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPatch, "/live", nil)
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("patch live: %d %s", w.Code, w.Body.String())
	}
}

func coverNavBranches(t *testing.T, app *App, admin string) {
	t.Helper()
	if w := doJSON(t, app, http.MethodGet, "/api/v1/nav-menus?audience=web", admin, nil); w.Code != http.StatusOK {
		t.Fatalf("nav web: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/nav-menus?audience=nope", admin, nil); w.Code != http.StatusBadRequest {
		t.Fatalf("nav bad audience: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPut, "/api/v1/nav-menus/99999", admin, map[string]any{"name": "x"}); w.Code != http.StatusNotFound {
		t.Fatalf("nav missing: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPut, "/api/v1/nav-menus/reorder", admin, map[string]any{"items": []any{}}); w.Code != http.StatusBadRequest {
		t.Fatalf("reorder empty: %d", w.Code)
	}
	parent := doJSON(t, app, http.MethodPost, "/api/v1/nav-menus", admin, map[string]any{
		"audience": "admin", "name": "CovP", "code": "cov:parent", "routePath": "/cov-p",
		"component": "DashboardPage", "icon": "Radio", "sort": 80, "permCode": "dashboard:read",
	})
	if parent.Code != http.StatusOK {
		t.Fatalf("nav parent: %d %s", parent.Code, parent.Body.String())
	}
	var p models.NavMenu
	if err := json.Unmarshal(decodeEnv(t, parent).Data, &p); err != nil {
		t.Fatal(err)
	}
	hidden := true
	sort := 81
	child := doJSON(t, app, http.MethodPost, "/api/v1/nav-menus", admin, map[string]any{
		"parentId": p.ID, "audience": "admin", "name": "CovC", "code": "cov:child",
		"routePath": "/cov-c", "component": "DashboardPage", "icon": "Radio", "sort": sort,
		"hidden": hidden, "permCode": "dashboard:read", "status": "active",
	})
	if child.Code != http.StatusOK {
		t.Fatalf("nav child: %d %s", child.Code, child.Body.String())
	}
	var ch models.NavMenu
	if err := json.Unmarshal(decodeEnv(t, child).Data, &ch); err != nil {
		t.Fatal(err)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/nav-menus", admin, map[string]any{
		"name": "Dup", "code": "cov:parent",
	}); w.Code != http.StatusConflict {
		t.Fatalf("nav dup: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodPut, "/api/v1/nav-menus/"+itoa(p.ID), admin, map[string]any{
		"name": "CovP2", "routePath": "/cov-p2", "component": "UsersPage", "icon": "Users",
		"permCode": "user:list", "status": "active", "hidden": false, "sort": 79,
	}); w.Code != http.StatusOK {
		t.Fatalf("nav upd: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodPut, "/api/v1/nav-menus/reorder", admin, map[string]any{
		"items": []map[string]any{{"id": p.ID, "sort": 70}, {"id": ch.ID, "sort": 71, "parentId": p.ID}},
	}); w.Code != http.StatusOK {
		t.Fatalf("reorder: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodDelete, "/api/v1/nav-menus/"+itoa(p.ID), admin, nil); w.Code != http.StatusBadRequest {
		t.Fatalf("delete parent with child: %d %s", w.Code, w.Body.String())
	}
	var sys models.NavMenu
	if err := app.DB.Where("is_system = ?", true).First(&sys).Error; err == nil {
		if w := doJSON(t, app, http.MethodDelete, "/api/v1/nav-menus/"+itoa(sys.ID), admin, nil); w.Code != http.StatusBadRequest {
			t.Fatalf("delete system: %d %s", w.Code, w.Body.String())
		}
	}
	if w := doJSON(t, app, http.MethodDelete, "/api/v1/nav-menus/"+itoa(ch.ID), admin, nil); w.Code != http.StatusOK {
		t.Fatalf("del child: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodDelete, "/api/v1/nav-menus/"+itoa(p.ID), admin, nil); w.Code != http.StatusOK {
		t.Fatalf("del parent: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodDelete, "/api/v1/nav-menus/99999", admin, nil); w.Code != http.StatusNotFound {
		t.Fatalf("del missing: %d", w.Code)
	}
}

func coverMailAndLogs(t *testing.T, app *App, admin string) {
	t.Helper()
	camp := doJSON(t, app, http.MethodPost, "/api/v1/mail/campaigns", admin, map[string]string{
		"name": "cov2", "subject": "hi", "body": "<p>x</p>", "audience": "all_active",
	})
	if camp.Code != http.StatusOK {
		t.Fatalf("camp: %d %s", camp.Code, camp.Body.String())
	}
	var row models.MailCampaign
	if err := json.Unmarshal(decodeEnv(t, camp).Data, &row); err != nil {
		t.Fatal(err)
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/mail/campaigns", admin, nil); w.Code != http.StatusOK {
		t.Fatalf("list camp: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/mail/campaigns/"+itoa(row.ID), admin, nil); w.Code != http.StatusOK {
		t.Fatalf("get camp: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/mail/campaigns/0", admin, nil); w.Code != http.StatusNotFound {
		t.Fatalf("get camp 0: %d", w.Code)
	}
	name, subject, body, audience := "cov2b", "hi2", "<p>y</p>", "opted_in"
	if w := doJSON(t, app, http.MethodPut, "/api/v1/mail/campaigns/"+itoa(row.ID), admin, map[string]any{
		"name": &name, "subject": &subject, "body": &body, "audience": &audience,
	}); w.Code != http.StatusOK {
		t.Fatalf("upd camp: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodPut, "/api/v1/mail/campaigns/"+itoa(row.ID), admin, map[string]any{
		"audience": "bad",
	}); w.Code != http.StatusBadRequest {
		t.Fatalf("camp audience: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/mail/campaigns/"+itoa(row.ID)+"/schedule", admin, map[string]any{}); w.Code != http.StatusOK {
		t.Fatalf("schedule: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodDelete, "/api/v1/mail/campaigns/"+itoa(row.ID), admin, nil); w.Code != http.StatusBadRequest {
		t.Fatalf("delete scheduled: %d", w.Code)
	}
	paused := models.CampaignPaused
	if w := doJSON(t, app, http.MethodPut, "/api/v1/mail/campaigns/"+itoa(row.ID), admin, map[string]any{"status": &paused}); w.Code != http.StatusOK {
		t.Fatalf("pause: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodDelete, "/api/v1/mail/campaigns/"+itoa(row.ID), admin, nil); w.Code != http.StatusOK {
		t.Fatalf("del paused: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/mail/jobs/999/retry", admin, nil); w.Code != http.StatusNotFound {
		t.Fatalf("retry missing: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/mail/jobs/abc/cancel", admin, nil); w.Code != http.StatusNotFound {
		t.Fatalf("cancel abc: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/mail/jobs?status=queued&class=transactional", admin, nil); w.Code != http.StatusOK {
		t.Fatalf("jobs: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodDelete, "/api/v1/logs?kind=login", admin, nil); w.Code != http.StatusOK {
		t.Fatalf("clear login: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodDelete, "/api/v1/logs?kind=api", admin, nil); w.Code != http.StatusOK {
		t.Fatalf("clear api: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/logs/purge?days=1", admin, nil); w.Code != http.StatusOK {
		t.Fatalf("purge: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/logs", admin, nil); w.Code != http.StatusOK {
		t.Fatalf("logs: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/logs/login", admin, nil); w.Code != http.StatusOK {
		t.Fatalf("login logs: %d", w.Code)
	}
}

func coverDepartmentsAndUsers(t *testing.T, app *App, admin string) {
	t.Helper()
	var hq models.Department
	if err := app.DB.Where("code = ?", "hq").First(&hq).Error; err != nil {
		t.Fatal(err)
	}
	created := doJSON(t, app, http.MethodPost, "/api/v1/departments", admin, map[string]any{
		"name": "覆盖部", "code": "cov-dept", "parentId": hq.ID, "sort": 9, "leader": "x", "status": "active",
	})
	if created.Code != http.StatusOK {
		t.Fatalf("dept: %d %s", created.Code, created.Body.String())
	}
	var dept models.Department
	if err := json.Unmarshal(decodeEnv(t, created).Data, &dept); err != nil {
		t.Fatal(err)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/departments", admin, map[string]any{
		"name": "覆盖部", "code": "hq",
	}); w.Code != http.StatusConflict {
		t.Fatalf("dept dup: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodPut, "/api/v1/departments/"+itoa(dept.ID), admin, map[string]any{
		"name": "覆盖部2", "code": "cov-dept", "parentId": hq.ID, "leader": "y", "status": "active",
	}); w.Code != http.StatusOK {
		t.Fatalf("upd dept: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodPut, "/api/v1/departments/"+itoa(dept.ID), admin, map[string]any{
		"parentId": dept.ID,
	}); w.Code != http.StatusBadRequest {
		t.Fatalf("dept cycle: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodDelete, "/api/v1/departments/"+itoa(hq.ID), admin, nil); w.Code != http.StatusBadRequest {
		t.Fatalf("del hq with child: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodDelete, "/api/v1/departments/"+itoa(dept.ID), admin, nil); w.Code != http.StatusOK {
		t.Fatalf("del dept: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/departments?q=hq", admin, nil); w.Code != http.StatusOK {
		t.Fatalf("list dept: %d", w.Code)
	}

	var viewerRole models.Role
	if err := app.DB.Where("code = ?", seed.RoleViewer).First(&viewerRole).Error; err != nil {
		t.Fatal(err)
	}
	user := doJSON(t, app, http.MethodPost, "/api/v1/users", admin, map[string]any{
		"username": "cov-boost", "password": "cov-boost-1a", "nickname": "Boost",
		"email": "cov-boost@example.com", "phone": "13900000099", "gender": "male",
		"department": "tech", "roleIds": []uint{viewerRole.ID},
	})
	if user.Code != http.StatusOK {
		t.Fatalf("user: %d %s", user.Code, user.Body.String())
	}
	var createdUser userDTO
	if err := json.Unmarshal(decodeEnv(t, user).Data, &createdUser); err != nil {
		t.Fatal(err)
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/users/"+itoa(createdUser.ID), admin, nil); w.Code != http.StatusOK {
		t.Fatalf("get user: %d", w.Code)
	}
	nick, phone, gender, deptCode, title, remark, tz := "Boost2", "13900000098", "female", "ops", "eng", "note", "Asia/Shanghai"
	optIn := true
	if w := doJSON(t, app, http.MethodPut, "/api/v1/users/"+itoa(createdUser.ID), admin, map[string]any{
		"nickname": &nick, "phone": &phone, "gender": &gender, "department": &deptCode,
		"title": &title, "remark": &remark, "timezone": &tz, "marketingOptIn": &optIn,
	}); w.Code != http.StatusOK {
		t.Fatalf("upd user: %d %s", w.Code, w.Body.String())
	}
	badAvatar := "/tmp/x.png"
	if w := doJSON(t, app, http.MethodPut, "/api/v1/users/"+itoa(createdUser.ID), admin, map[string]any{
		"avatar": &badAvatar,
	}); w.Code != http.StatusBadRequest {
		t.Fatalf("bad avatar: %d", w.Code)
	}
	badTZ := "Not/AZone"
	if w := doJSON(t, app, http.MethodPut, "/api/v1/users/"+itoa(createdUser.ID), admin, map[string]any{
		"timezone": &badTZ,
	}); w.Code != http.StatusBadRequest {
		t.Fatalf("bad tz: %d", w.Code)
	}
	taken := "admin@latch.local"
	if w := doJSON(t, app, http.MethodPut, "/api/v1/users/"+itoa(createdUser.ID), admin, map[string]any{
		"email": &taken,
	}); w.Code != http.StatusConflict {
		t.Fatalf("email taken: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/users?q=cov-boost&kind=admin", admin, nil); w.Code != http.StatusOK {
		t.Fatalf("list users: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPut, "/api/v1/users/"+itoa(createdUser.ID)+"/roles", admin, map[string]any{
		"roleIds": []uint{viewerRole.ID},
	}); w.Code != http.StatusOK {
		t.Fatalf("assign roles: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/users/"+itoa(createdUser.ID)+"/revoke", admin, nil); w.Code != http.StatusOK {
		t.Fatalf("revoke: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodDelete, "/api/v1/users/"+itoa(createdUser.ID), admin, nil); w.Code != http.StatusOK {
		t.Fatalf("del user: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodDelete, "/api/v1/users/1", admin, nil); w.Code != http.StatusBadRequest {
		t.Fatalf("del seed: %d", w.Code)
	}
}

func coverRolesPermsAnnounce(t *testing.T, app *App, admin string) {
	t.Helper()
	role := doJSON(t, app, http.MethodPost, "/api/v1/roles", admin, map[string]any{
		"name": "cov-role-2", "code": "cov-role-2", "dataScope": "dept_sub", "description": "x",
		"permissionIds": []uint{},
	})
	if role.Code != http.StatusOK {
		t.Fatalf("role: %d %s", role.Code, role.Body.String())
	}
	var created roleDTO
	if err := json.Unmarshal(decodeEnv(t, role).Data, &created); err != nil {
		t.Fatal(err)
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/roles/"+itoa(created.ID), admin, nil); w.Code != http.StatusOK {
		t.Fatalf("get role: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/roles/99999", admin, nil); w.Code != http.StatusNotFound {
		t.Fatalf("role missing: %d", w.Code)
	}
	name, desc, scope := "cov-role-2b", "y", models.DataScopeDept
	if w := doJSON(t, app, http.MethodPut, "/api/v1/roles/"+itoa(created.ID), admin, map[string]any{
		"name": &name, "description": &desc, "dataScope": &scope,
	}); w.Code != http.StatusOK {
		t.Fatalf("upd role: %d %s", w.Code, w.Body.String())
	}
	badScope := "galaxy"
	if w := doJSON(t, app, http.MethodPut, "/api/v1/roles/"+itoa(created.ID), admin, map[string]any{
		"dataScope": &badScope,
	}); w.Code != http.StatusBadRequest {
		t.Fatalf("bad scope: %d", w.Code)
	}
	perm := doJSON(t, app, http.MethodPost, "/api/v1/permissions", admin, map[string]any{
		"name": "cov perm 2", "code": "cov:perm2", "path": "/api/v1/cov2", "method": "GET", "kind": "api",
	})
	if perm.Code != http.StatusOK {
		t.Fatalf("perm: %d %s", perm.Code, perm.Body.String())
	}
	var createdPerm permissionDTO
	if err := json.Unmarshal(decodeEnv(t, perm).Data, &createdPerm); err != nil {
		t.Fatal(err)
	}
	if w := doJSON(t, app, http.MethodPut, "/api/v1/roles/"+itoa(created.ID)+"/permissions", admin, map[string]any{
		"permissionIds": []uint{createdPerm.ID},
	}); w.Code != http.StatusOK {
		t.Fatalf("assign perm: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodPut, "/api/v1/permissions/"+itoa(createdPerm.ID), admin, map[string]any{
		"name": "cov perm 2b", "code": "cov:perm2", "path": "/api/v1/cov2", "method": "GET", "kind": "api",
	}); w.Code != http.StatusOK {
		t.Fatalf("upd assigned perm: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/roles/"+itoa(created.ID)+"/copy", admin, map[string]any{
		"name": "cov-role-2-copy", "code": "cov-role-2-copy",
	}); w.Code != http.StatusOK {
		t.Fatalf("copy: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/roles/"+itoa(created.ID)+"/copy", admin, map[string]any{
		"name": "admin", "code": "admin",
	}); w.Code != http.StatusBadRequest {
		t.Fatalf("copy builtin: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/roles/"+itoa(created.ID)+"/copy", admin, map[string]any{
		"name": "again", "code": "cov-role-2-copy",
	}); w.Code != http.StatusConflict {
		t.Fatalf("copy dup: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodDelete, "/api/v1/permissions/"+itoa(createdPerm.ID), admin, nil); w.Code != http.StatusOK {
		t.Fatalf("del used perm: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodDelete, "/api/v1/roles/1", admin, nil); w.Code != http.StatusBadRequest {
		t.Fatalf("del seed role: %d", w.Code)
	}
	var copy roleDTO
	listed := decodeRolePage(t, doJSON(t, app, http.MethodGet, "/api/v1/roles?q=cov-role-2-copy", admin, nil))
	for _, r := range listed {
		if r.Code == "cov-role-2-copy" {
			copy = r
		}
	}
	if copy.ID != 0 {
		if w := doJSON(t, app, http.MethodDelete, "/api/v1/roles/"+itoa(copy.ID), admin, nil); w.Code != http.StatusOK {
			t.Fatalf("del copy: %d %s", w.Code, w.Body.String())
		}
	}
	if w := doJSON(t, app, http.MethodDelete, "/api/v1/roles/"+itoa(created.ID), admin, nil); w.Code != http.StatusOK {
		t.Fatalf("del role: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/announcements", admin, map[string]string{
		"kind": "admin", "title": "",
	}); w.Code != http.StatusBadRequest {
		t.Fatalf("announce title: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/announcements", admin, map[string]string{
		"kind": "admin", "title": "cov announce", "body": "hello",
	}); w.Code != http.StatusOK {
		t.Fatalf("announce: %d %s", w.Code, w.Body.String())
	}
	box := decodePage[models.Notification](t, doJSON(t, app, http.MethodGet, "/api/v1/notifications?unread=1", admin, nil))
	if len(box.Items) > 0 {
		if w := doJSON(t, app, http.MethodPost, "/api/v1/notifications/"+formatUint(box.Items[0].ID)+"/read", admin, nil); w.Code != http.StatusOK {
			t.Fatalf("read one: %d", w.Code)
		}
	}
}

func coverImportErrors(t *testing.T, app *App) {
	t.Helper()
	csvBody := "username,password,nickname,email,phone,status,department,kind,gender\n" +
		",missing,,,,active,,admin,\n" +
		"badkind,badkind12a,,,,active,,nope,\n" +
		"weak,123,,,,active,,admin,\n" +
		"baddict,baddict12a,,,,zzz,,admin,\n" +
		"baddept,baddept12a,,,,active,no-such,admin,\n" +
		"dupmail,dupmail12a,,,admin@latch.local,active,,admin,\n" +
		"admin,admin1234a,,,,active,,admin,\n" +
		"okboost,okboost12a,OK,okboost@example.com,,active,tech,admin,male\n"
	created, failed, total, errs := app.importUsersCSV([]byte(csvBody), models.UserKindAdmin)
	if created == 0 || failed == 0 || total == 0 || errs == "" {
		t.Fatalf("import csv created=%d failed=%d total=%d errs=%q", created, failed, total, errs)
	}
	_, _, _, _ = app.importUsersCSV([]byte("not,csv"), models.UserKindAdmin)
	_, _, _, _ = app.importUsersCSV(nil, models.UserKindAdmin)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	missing := doJSON(t, app, http.MethodPost, "/api/v1/users/import", admin, nil)
	if missing.Code != http.StatusBadRequest {
		t.Fatalf("import no file: %d", missing.Code)
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/users/import-jobs/0", admin, nil); w.Code != http.StatusNotFound {
		t.Fatalf("job 0: %d", w.Code)
	}
}

func coverTotpLogin(t *testing.T, app *App, operator string) {
	t.Helper()
	setup := doJSON(t, app, http.MethodPost, "/api/v1/auth/totp/setup", operator, map[string]string{})
	if setup.Code != http.StatusOK {
		t.Fatalf("setup: %d %s", setup.Code, setup.Body.String())
	}
	var enroll struct {
		Secret string `json:"secret"`
		Ticket string `json:"ticket"`
	}
	if err := json.Unmarshal(decodeEnv(t, setup).Data, &enroll); err != nil {
		t.Fatal(err)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/totp/confirm", operator, map[string]string{
		"ticket": enroll.Ticket, "code": "000000",
	}); w.Code != http.StatusBadRequest {
		t.Fatalf("bad confirm: %d", w.Code)
	}
	code, err := totp.Code(enroll.Secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	confirm := doJSON(t, app, http.MethodPost, "/api/v1/auth/totp/confirm", operator, map[string]string{
		"ticket": enroll.Ticket, "code": code,
	})
	if confirm.Code != http.StatusOK {
		t.Fatalf("confirm: %d %s", confirm.Code, confirm.Body.String())
	}
	var enabled struct {
		RecoveryCodes []string `json:"recoveryCodes"`
	}
	if err := json.Unmarshal(decodeEnv(t, confirm).Data, &enabled); err != nil || len(enabled.RecoveryCodes) == 0 {
		t.Fatalf("recovery: %s", confirm.Body.String())
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/totp/setup", operator, map[string]string{}); w.Code != http.StatusBadRequest {
		t.Fatalf("already enabled: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/totp/recovery", operator, map[string]string{"code": "000000"}); w.Code != http.StatusBadRequest {
		t.Fatalf("bad recovery regen: %d", w.Code)
	}
	code2, err := totp.Code(enroll.Secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	regen := doJSON(t, app, http.MethodPost, "/api/v1/auth/totp/recovery", operator, map[string]string{"code": code2})
	if regen.Code != http.StatusOK {
		t.Fatalf("regen: %d %s", regen.Code, regen.Body.String())
	}
	if err := json.Unmarshal(decodeEnv(t, regen).Data, &enabled); err != nil || len(enabled.RecoveryCodes) == 0 {
		t.Fatalf("regen data: %s", regen.Body.String())
	}

	id, answer, _ := issueCaptcha(t, app)
	stepW := login(t, app, seed.OperatorUsername, seed.OperatorPassword, id, answer)
	if stepW.Code != http.StatusOK {
		t.Fatalf("login totp: %d %s", stepW.Code, stepW.Body.String())
	}
	var step struct {
		TotpRequired bool   `json:"totpRequired"`
		TotpTicket   string `json:"totpTicket"`
		Token        string `json:"token"`
	}
	if err := json.Unmarshal(decodeEnv(t, stepW).Data, &step); err != nil || !step.TotpRequired || step.TotpTicket == "" || step.Token != "" {
		t.Fatalf("expected totp step: %s", stepW.Body.String())
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/totp/verify", "", map[string]string{}); w.Code != http.StatusBadRequest {
		t.Fatalf("verify empty: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/totp/verify", "", map[string]string{
		"ticket": "nope", "code": "000000",
	}); w.Code != http.StatusUnauthorized {
		t.Fatalf("verify bad ticket: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/totp/verify", "", map[string]string{
		"ticket": step.TotpTicket, "code": "000000",
	}); w.Code != http.StatusBadRequest {
		t.Fatalf("verify bad code: %d", w.Code)
	}
	code3, err := totp.Code(enroll.Secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	verified := doJSON(t, app, http.MethodPost, "/api/v1/auth/totp/verify", "", map[string]string{
		"ticket": step.TotpTicket, "code": code3,
	})
	if verified.Code != http.StatusOK {
		t.Fatalf("verify: %d %s", verified.Code, verified.Body.String())
	}

	id2, answer2, _ := issueCaptcha(t, app)
	step2 := login(t, app, seed.OperatorUsername, seed.OperatorPassword, id2, answer2)
	var stepB struct {
		TotpTicket string `json:"totpTicket"`
	}
	if err := json.Unmarshal(decodeEnv(t, step2).Data, &stepB); err != nil || stepB.TotpTicket == "" {
		t.Fatalf("second totp step: %s", step2.Body.String())
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/totp/verify", "", map[string]string{
		"ticket": stepB.TotpTicket, "recoveryCode": "nope",
	}); w.Code != http.StatusBadRequest {
		t.Fatalf("bad recovery login: %d", w.Code)
	}
	recovered := doJSON(t, app, http.MethodPost, "/api/v1/auth/totp/verify", "", map[string]string{
		"ticket": stepB.TotpTicket, "recoveryCode": enabled.RecoveryCodes[0],
	})
	if recovered.Code != http.StatusOK {
		t.Fatalf("recovery login: %d %s", recovered.Code, recovered.Body.String())
	}
	var recTok struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(decodeEnv(t, recovered).Data, &recTok); err != nil || recTok.Token == "" {
		t.Fatalf("recovery token: %s", recovered.Body.String())
	}
	code4, err := totp.Code(enroll.Secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/totp/disable", recTok.Token, map[string]string{"code": code4}); w.Code != http.StatusOK {
		t.Fatalf("disable: %d %s", w.Code, w.Body.String())
	}
}

func coverAvatarForUser(t *testing.T, app *App, admin string) {
	t.Helper()
	user := doJSON(t, app, http.MethodPost, "/api/v1/users", admin, map[string]any{
		"username": "cov-ava", "password": "cov-ava-12a", "roleIds": []uint{},
	})
	if user.Code != http.StatusOK {
		t.Fatalf("ava user: %d %s", user.Code, user.Body.String())
	}
	var created userDTO
	if err := json.Unmarshal(decodeEnv(t, user).Data, &created); err != nil {
		t.Fatal(err)
	}
	posted := doMultipart(t, app, "/api/v1/users/"+itoa(created.ID)+"/avatar", admin, "file", "dot.png", tinyPNG())
	if posted.Code != http.StatusOK {
		t.Fatalf("user avatar: %d %s", posted.Code, posted.Body.String())
	}
	again := doMultipart(t, app, "/api/v1/users/"+itoa(created.ID)+"/avatar", admin, "file", "dot.png", tinyPNG())
	if again.Code != http.StatusOK {
		t.Fatalf("replace avatar: %d %s", again.Code, again.Body.String())
	}
	if w := doJSON(t, app, http.MethodDelete, "/api/v1/users/"+itoa(created.ID), admin, nil); w.Code != http.StatusOK {
		t.Fatalf("del ava: %d", w.Code)
	}
}
