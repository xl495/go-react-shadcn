package httpserver

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/googleid"
	"go-react-shadcn/internal/mailer"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/seed"
)

func TestGzipWriterWriteStringAndFlush(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	gz := gzip.NewWriter(rec)
	gw := gzipWriter{ResponseWriter: c.Writer, writer: gz}
	if _, err := gw.WriteString("abc"); err != nil {
		t.Fatal(err)
	}
	gw.Flush()
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestCoverageBoostFinalGaps(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)

	if w := doJSON(t, app, http.MethodGet, "/api/v1/users?status=active&gender=male&department=tech&roleId=1&sortBy=username", admin, nil); w.Code != http.StatusOK {
		t.Fatalf("list filters: %d %s", w.Code, w.Body.String())
	}

	u := doJSON(t, app, http.MethodPost, "/api/v1/users", admin, map[string]any{
		"username": "cov-final", "password": "cov-final-1a", "roleIds": []uint{},
	})
	if u.Code != http.StatusOK {
		t.Fatalf("user: %d %s", u.Code, u.Body.String())
	}
	var created userDTO
	if err := json.Unmarshal(decodeEnv(t, u).Data, &created); err != nil {
		t.Fatal(err)
	}
	pwd, status := "cov-final-2b", "disabled"
	if w := doJSON(t, app, http.MethodPut, "/api/v1/users/"+itoa(created.ID), admin, map[string]any{
		"password": &pwd, "status": &status,
	}); w.Code != http.StatusOK {
		t.Fatalf("upd pwd/status: %d %s", w.Code, w.Body.String())
	}

	var gender models.DictType
	if err := app.DB.Where("code = ?", seed.DictGender).First(&gender).Error; err != nil {
		t.Fatal(err)
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/dicts/"+itoa(gender.ID)+"/items", admin, nil); w.Code != http.StatusOK {
		t.Fatalf("list items: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/dicts/99999/items", admin, map[string]any{"label": "A", "value": "a"}); w.Code != http.StatusNotFound {
		t.Fatalf("item type miss: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/dicts/"+itoa(gender.ID)+"/items", admin, map[string]any{}); w.Code != http.StatusBadRequest {
		t.Fatalf("item empty: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPut, "/api/v1/dict-items/99999", admin, map[string]any{"label": "A"}); w.Code != http.StatusNotFound {
		t.Fatalf("item miss: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodDelete, "/api/v1/dict-items/99999", admin, nil); w.Code != http.StatusNotFound {
		t.Fatalf("item del miss: %d", w.Code)
	}

	if w := doJSON(t, app, http.MethodPut, "/api/v1/auth/profile", admin, map[string]string{"email": ""}); w.Code != http.StatusOK {
		t.Fatalf("clear email: %d %s", w.Code, w.Body.String())
	}

	app.Cfg.DevMode = false
	if w := doJSON(t, app, http.MethodPost, "/api/v1/users", admin, map[string]any{
		"username": "seedpw", "password": seed.AdminPassword,
	}); w.Code != http.StatusBadRequest {
		t.Fatalf("seed pwd: %d %s", w.Code, w.Body.String())
	}
	app.Cfg.DevMode = true

	setup := doJSON(t, app, http.MethodPost, "/api/v1/auth/totp/setup", admin, map[string]string{})
	if setup.Code != http.StatusOK {
		t.Fatalf("totp setup: %d %s", setup.Code, setup.Body.String())
	}
	var enroll struct {
		Secret string `json:"secret"`
		Ticket string `json:"ticket"`
	}
	if err := json.Unmarshal(decodeEnv(t, setup).Data, &enroll); err != nil {
		t.Fatal(err)
	}
	again := doJSON(t, app, http.MethodPost, "/api/v1/auth/totp/setup", admin, map[string]string{"ticket": enroll.Ticket})
	if again.Code != http.StatusOK {
		t.Fatalf("totp setup ticket: %d %s", again.Code, again.Body.String())
	}
	if err := json.Unmarshal(decodeEnv(t, again).Data, &enroll); err != nil {
		t.Fatal(err)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/totp/confirm", admin, map[string]string{"code": "123456"}); w.Code != http.StatusUnauthorized {
		t.Fatalf("confirm no ticket: %d", w.Code)
	}

	var hq models.Department
	if err := app.DB.Where("code = ?", "hq").First(&hq).Error; err != nil {
		t.Fatal(err)
	}
	scopes := []models.User{
		{ID: 1, Kind: models.UserKindAdmin, Roles: []models.Role{{DataScope: models.DataScopeSelf}}},
		{ID: 1, Kind: models.UserKindWeb, Roles: []models.Role{{DataScope: models.DataScopeSelf}}},
		{ID: 1, Kind: models.UserKindAdmin, Roles: []models.Role{{DataScope: models.DataScopeDept}}},
		{ID: 1, Kind: models.UserKindAdmin, DepartmentID: &hq.ID, Roles: []models.Role{{DataScope: models.DataScopeDept}}},
		{ID: 1, Kind: models.UserKindAdmin, Roles: []models.Role{{DataScope: models.DataScopeDeptAndSub}}},
		{ID: 1, Kind: models.UserKindAdmin, DepartmentID: &hq.ID, Roles: []models.Role{{DataScope: models.DataScopeDeptAndSub}}},
		{ID: 1, Kind: models.UserKindAdmin, Roles: []models.Role{{DataScope: "custom"}}},
	}
	for _, user := range scopes {
		var rows []models.User
		_ = app.applyUserDataScope(app.accounts(models.UserKindAdmin), user, models.UserKindAdmin).Find(&rows).Error
	}

	if w := doJSON(t, app, http.MethodDelete, "/api/v1/users/"+itoa(created.ID), admin, nil); w.Code != http.StatusOK {
		t.Fatalf("del final: %d", w.Code)
	}

	_ = collectCodes(models.User{Roles: []models.Role{{
		Code: "x", Permissions: []models.Permission{{Code: ""}, {Code: "a"}, {Code: "a"}},
	}}})

	app.totpTickets = nil
	var adminUser models.User
	if err := app.accounts(models.UserKindAdmin).Where("username = ?", seed.AdminUsername).First(&adminUser).Error; err != nil {
		t.Fatal(err)
	}
	tid := app.issueTotpTicket(adminUser, "login", "")
	app.sweepTotpTickets(time.Now().Add(time.Hour))
	app.putTotpTicket("expired", totpTicket{userID: adminUser.ID, kind: adminUser.Kind, purpose: "login", expires: time.Now().Add(-time.Minute)})
	if _, ok := app.takeTotpTicket("expired", "login"); ok {
		t.Fatal("expired ticket")
	}
	_ = tid

	if w := doJSON(t, app, http.MethodPost, "/api/v1/mail/unsubscribe", "", "nope"); w.Code != http.StatusBadRequest {
		t.Fatalf("unsub bind: %d", w.Code)
	}
	missing := mailer.UnsubToken(app.Cfg.UnsubSecret(), models.UserKindAdmin, 99999)
	if w := doJSON(t, app, http.MethodPost, "/api/v1/mail/unsubscribe", "", map[string]string{"token": missing}); w.Code != http.StatusNotFound {
		t.Fatalf("unsub missing: %d %s", w.Code, w.Body.String())
	}
	real := mailer.UnsubToken(app.Cfg.UnsubSecret(), models.UserKindAdmin, adminUser.ID)
	if w := doJSON(t, app, http.MethodPost, "/api/v1/mail/unsubscribe", "", map[string]string{"token": real}); w.Code != http.StatusOK {
		t.Fatalf("unsub ok: %d %s", w.Code, w.Body.String())
	}

	setCfg(t, app, "app.maintenance", "1")
	id, answer, _ := issueCaptcha(t, app)
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/register", "", map[string]string{
		"username": "maintuser", "email": "maint@example.com", "password": "maintuser12",
		"client": "web", "captchaId": id, "captchaCode": answer,
	}); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("register maint: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/google", "", map[string]string{
		"idToken": "x", "client": "web",
	}); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("google maint: %d", w.Code)
	}
	setCfg(t, app, "app.maintenance", "0")

	if w := doJSON(t, app, http.MethodPost, "/api/v1/roles/1/copy", admin, map[string]any{}); w.Code != http.StatusBadRequest {
		t.Fatalf("copy empty: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/google", "", "nope"); w.Code != http.StatusBadRequest {
		t.Fatalf("google bind json: %d", w.Code)
	}

	app.notify(models.UserKindAdmin, 0, "x", "t", "", "", 0)
	app.totpTickets = nil
	app.putTotpTicket("z", totpTicket{expires: time.Now().Add(time.Minute)})
	_ = app.sysOn("definitely.missing.cfg", true)
	var nilApp *App
	nilApp.warnSealedConfigs()
}

func TestCoverageBoostValidationAndGoogle(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)

	coverRegisterAndVerify(t, app)
	coverCreateUserErrors(t, app, admin)
	coverUploadErrors(t, app, admin)
	coverConfigDictErrors(t, app, admin)
	coverAuthProfilePassword(t, app, admin)
	coverGoogleBranches(t, app, admin)
	coverMustChangeExtras(t, app, admin)
	coverMiscLookups(t, app, admin)
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/logout", admin, nil); w.Code != http.StatusOK {
		t.Fatalf("logout: %d %s", w.Code, w.Body.String())
	}
}

func coverRegisterAndVerify(t *testing.T, app *App) {
	t.Helper()
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/register", "", "nope"); w.Code != http.StatusBadRequest {
		t.Fatalf("register bind: %d", w.Code)
	}
	id, answer, _ := issueCaptcha(t, app)
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/register", "", map[string]string{
		"username": "", "email": "", "password": "", "client": "web", "captchaId": id, "captchaCode": answer,
	}); w.Code != http.StatusBadRequest {
		t.Fatalf("register empty: %d", w.Code)
	}
	id, answer, _ = issueCaptcha(t, app)
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/register", "", map[string]string{
		"username": "weakone", "email": "weakone@example.com", "password": "123",
		"client": "web", "captchaId": id, "captchaCode": answer,
	}); w.Code != http.StatusBadRequest {
		t.Fatalf("register weak: %d", w.Code)
	}
	id, answer, _ = issueCaptcha(t, app)
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/register", "", map[string]string{
		"username": seed.MemberUsername, "email": "other@example.com", "password": "member-new-12",
		"client": "web", "captchaId": id, "captchaCode": answer,
	}); w.Code != http.StatusConflict {
		t.Fatalf("register dup user: %d %s", w.Code, w.Body.String())
	}
	id, answer, _ = issueCaptcha(t, app)
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/register", "", map[string]string{
		"username": "newmailuser", "email": "webuser@latch.local", "password": "newmailuser12",
		"client": "web", "captchaId": id, "captchaCode": answer,
	}); w.Code != http.StatusConflict {
		t.Fatalf("register dup email: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/verify-email", "", map[string]string{}); w.Code != http.StatusBadRequest {
		t.Fatalf("verify empty: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/verify-email", "", map[string]string{"token": "nope"}); w.Code != http.StatusBadRequest {
		t.Fatalf("verify bad: %d", w.Code)
	}
}

func coverCreateUserErrors(t *testing.T, app *App, admin string) {
	t.Helper()
	if w := doJSON(t, app, http.MethodPost, "/api/v1/users", admin, map[string]any{}); w.Code != http.StatusBadRequest {
		t.Fatalf("user empty: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/users", admin, map[string]any{
		"username": "shortpw", "password": "1",
	}); w.Code != http.StatusBadRequest {
		t.Fatalf("user weak: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/users", admin, map[string]any{
		"username": "badkind", "password": "badkind12a", "kind": "robot",
	}); w.Code != http.StatusBadRequest {
		t.Fatalf("user kind: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/users", admin, map[string]any{
		"username": "badrole", "password": "badrole12a", "roleIds": []uint{99999},
	}); w.Code != http.StatusBadRequest {
		t.Fatalf("user roles: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/users", admin, map[string]any{
		"username": "bademail", "password": "bademail12a", "email": "admin@latch.local",
	}); w.Code != http.StatusConflict {
		t.Fatalf("user email: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/users", admin, map[string]any{
		"username": seed.ViewerUsername, "password": "viewer-new-12a",
	}); w.Code != http.StatusConflict {
		t.Fatalf("user dup: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/users", admin, map[string]any{
		"username": "badtz", "password": "badtz12aaa", "timezone": "Nope/Zone",
	}); w.Code != http.StatusBadRequest {
		t.Fatalf("user tz: %d", w.Code)
	}
	opt := true
	web := doJSON(t, app, http.MethodPost, "/api/v1/users", admin, map[string]any{
		"username": "cov-web", "password": "cov-web-12a", "kind": "web",
		"email": "cov-web@example.com", "marketingOptIn": &opt, "timezone": "UTC",
	})
	if web.Code != http.StatusOK {
		t.Fatalf("web user: %d %s", web.Code, web.Body.String())
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/users?kind=web&q=cov-web", admin, nil); w.Code != http.StatusOK {
		t.Fatalf("list web: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/users/99999", admin, nil); w.Code != http.StatusNotFound {
		t.Fatalf("user miss: %d", w.Code)
	}
}

func coverUploadErrors(t *testing.T, app *App, admin string) {
	t.Helper()
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/avatar", admin, nil); w.Code != http.StatusBadRequest {
		t.Fatalf("own avatar none: %d", w.Code)
	}
	if w := doMultipart(t, app, "/api/v1/auth/avatar", admin, "file", "x.txt", []byte("hello")); w.Code != http.StatusBadRequest {
		t.Fatalf("bad type: %d", w.Code)
	}
	big := bytes.Repeat([]byte{0x89, 0x50, 0x4e, 0x47}, (2<<20)/4+8)
	if w := doMultipart(t, app, "/api/v1/auth/avatar", admin, "file", "big.png", big); w.Code != http.StatusBadRequest {
		t.Fatalf("too big: %d", w.Code)
	}
	if w := doMultipart(t, app, "/api/v1/users/99999/avatar", admin, "file", "dot.png", tinyPNG()); w.Code != http.StatusNotFound {
		t.Fatalf("avatar miss user: %d", w.Code)
	}
}

func coverConfigDictErrors(t *testing.T, app *App, admin string) {
	t.Helper()
	if w := doJSON(t, app, http.MethodPost, "/api/v1/configs", admin, map[string]string{}); w.Code != http.StatusBadRequest {
		t.Fatalf("cfg empty: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/configs", admin, map[string]string{
		"key": "app.name", "value": "x", "name": "n",
	}); w.Code != http.StatusConflict {
		t.Fatalf("cfg dup: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPut, "/api/v1/configs/99999", admin, map[string]string{"value": "x"}); w.Code != http.StatusNotFound {
		t.Fatalf("cfg miss: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodDelete, "/api/v1/configs/99999", admin, nil); w.Code != http.StatusNotFound {
		t.Fatalf("cfg del miss: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPut, "/api/v1/configs/batch", admin, map[string]any{"items": []any{}}); w.Code != http.StatusBadRequest {
		t.Fatalf("batch empty: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPut, "/api/v1/configs/batch", admin, map[string]any{
		"items": []map[string]any{{"key": "app.cov_batch", "value": "1", "name": "cov batch"}},
	}); w.Code != http.StatusOK {
		t.Fatalf("batch create: %d %s", w.Code, w.Body.String())
	}
	secret := doJSON(t, app, http.MethodPost, "/api/v1/configs", admin, map[string]string{
		"key": "mail.cov_secret", "value": "s3cret", "name": "secret", "group": "mail",
	})
	if secret.Code != http.StatusOK {
		t.Fatalf("secret cfg: %d %s", secret.Code, secret.Body.String())
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/configs?group=app&q=name", admin, nil); w.Code != http.StatusOK {
		t.Fatalf("list cfg: %d", w.Code)
	}
	prev := app.Cfg.DevMode
	app.Cfg.DevMode = false
	if w := doJSON(t, app, http.MethodPost, "/api/v1/configs", admin, map[string]string{
		"key": "auth.captcha_provider", "value": "none", "name": "cap",
	}); w.Code != http.StatusBadRequest {
		t.Fatalf("prod captcha: %d %s", w.Code, w.Body.String())
	}
	app.Cfg.DevMode = prev

	if w := doJSON(t, app, http.MethodPost, "/api/v1/dicts", admin, map[string]string{}); w.Code != http.StatusBadRequest {
		t.Fatalf("dict empty: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/dicts", admin, map[string]string{"code": "sys_gender", "name": "x"}); w.Code != http.StatusConflict {
		t.Fatalf("dict dup: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPut, "/api/v1/dicts/99999", admin, map[string]string{"name": "x"}); w.Code != http.StatusNotFound {
		t.Fatalf("dict miss: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodDelete, "/api/v1/dicts/99999", admin, nil); w.Code != http.StatusNotFound {
		t.Fatalf("dict del miss: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/dicts/99999/items", admin, nil); w.Code != http.StatusOK && w.Code != http.StatusNotFound {
		t.Fatalf("items miss: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/dicts?q=gender", admin, nil); w.Code != http.StatusOK {
		t.Fatalf("list dicts: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/roles", admin, map[string]string{}); w.Code != http.StatusBadRequest {
		t.Fatalf("role empty: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPut, "/api/v1/roles/99999", admin, map[string]any{"name": "x"}); w.Code != http.StatusNotFound {
		t.Fatalf("role miss: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/permissions", admin, map[string]any{}); w.Code != http.StatusBadRequest {
		t.Fatalf("perm empty: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPut, "/api/v1/permissions/99999", admin, map[string]any{"name": "x"}); w.Code != http.StatusNotFound {
		t.Fatalf("perm miss: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodDelete, "/api/v1/permissions/99999", admin, nil); w.Code != http.StatusNotFound {
		t.Fatalf("perm del miss: %d", w.Code)
	}
}

func coverAuthProfilePassword(t *testing.T, app *App, admin string) {
	t.Helper()
	id, answer, _ := issueCaptcha(t, app)
	if w := login(t, app, "", "", id, answer); w.Code != http.StatusUnauthorized {
		t.Fatalf("empty login: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPut, "/api/v1/auth/profile", admin, map[string]string{"gender": "alien"}); w.Code != http.StatusBadRequest {
		t.Fatalf("profile gender: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPut, "/api/v1/auth/profile", admin, map[string]string{"timezone": "Nope/Zone"}); w.Code != http.StatusBadRequest {
		t.Fatalf("profile tz: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPut, "/api/v1/auth/profile", admin, map[string]string{"email": "viewer@latch.local"}); w.Code != http.StatusConflict {
		t.Fatalf("profile taken: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodPut, "/api/v1/auth/password", admin, map[string]string{}); w.Code != http.StatusBadRequest {
		t.Fatalf("pwd empty: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPut, "/api/v1/auth/password", admin, map[string]string{
		"oldPassword": "wrong-pass-9", "newPassword": "admin-new-12a",
	}); w.Code != http.StatusBadRequest {
		t.Fatalf("pwd wrong: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPut, "/api/v1/auth/password", admin, map[string]string{
		"oldPassword": seed.AdminPassword, "newPassword": "123",
	}); w.Code != http.StatusBadRequest {
		t.Fatalf("pwd weak: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPut, "/api/v1/auth/password", admin, map[string]string{
		"oldPassword": seed.AdminPassword, "newPassword": seed.AdminUsername,
	}); w.Code != http.StatusBadRequest {
		t.Fatalf("pwd same user: %d", w.Code)
	}
}

func coverGoogleBranches(t *testing.T, app *App, admin string) {
	t.Helper()
	setCfg(t, app, "auth.google_enabled", "1")
	setCfg(t, app, "auth.google_client_id", "client-1")
	setCfg(t, app, "auth.google_register_enabled", "1")

	app.GoogleVerify = stubGoogle{err: errors.New("bad token")}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/google", "", map[string]string{"idToken": "x", "client": "web"}); w.Code != http.StatusUnauthorized {
		t.Fatalf("google bad token: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/google/bind", admin, map[string]string{}); w.Code != http.StatusBadRequest {
		t.Fatalf("bind empty: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/google/bind", admin, map[string]string{"idToken": "x"}); w.Code != http.StatusUnauthorized {
		t.Fatalf("bind bad token: %d", w.Code)
	}

	app.GoogleVerify = stubGoogle{ident: googleid.Identity{Subject: "g1", Email: "a@b.com", EmailVerified: false}}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/google", "", map[string]string{"idToken": "x", "client": "web"}); w.Code != http.StatusUnauthorized {
		t.Fatalf("google unverified: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/google/bind", admin, map[string]string{"idToken": "x"}); w.Code != http.StatusUnauthorized {
		t.Fatalf("bind unverified: %d", w.Code)
	}

	app.GoogleVerify = stubGoogle{ident: googleid.Identity{Subject: "g2", Email: "other@example.com", EmailVerified: true}}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/google/bind", admin, map[string]string{"idToken": "x"}); w.Code != http.StatusBadRequest {
		t.Fatalf("bind mismatch: %d %s", w.Code, w.Body.String())
	}

	if err := app.accounts(models.UserKindAdmin).Where("username = ?", seed.ViewerUsername).Update("google_id", "g-conflict").Error; err != nil {
		t.Fatal(err)
	}
	app.GoogleVerify = stubGoogle{ident: googleid.Identity{Subject: "g-conflict", Email: "admin@latch.local", EmailVerified: true}}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/google/bind", admin, map[string]string{"idToken": "x"}); w.Code != http.StatusConflict {
		t.Fatalf("bind conflict: %d %s", w.Code, w.Body.String())
	}

	until := time.Now().Add(time.Hour)
	if err := app.accounts(models.UserKindWeb).Where("username = ?", seed.MemberUsername).Updates(map[string]any{
		"google_id": "g-locked", "locked_until": until,
	}).Error; err != nil {
		t.Fatal(err)
	}
	app.GoogleVerify = stubGoogle{ident: googleid.Identity{Subject: "g-locked", Email: "webuser@latch.local", EmailVerified: true}}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/google", "", map[string]string{"idToken": "x", "client": "web"}); w.Code != http.StatusForbidden {
		t.Fatalf("google locked: %d %s", w.Code, w.Body.String())
	}
	if err := app.accounts(models.UserKindWeb).Where("username = ?", seed.MemberUsername).Updates(map[string]any{
		"locked_until": nil, "status": "disabled",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/google", "", map[string]string{"idToken": "x", "client": "web"}); w.Code != http.StatusUnauthorized {
		t.Fatalf("google disabled: %d %s", w.Code, w.Body.String())
	}
}

func coverMustChangeExtras(t *testing.T, app *App, admin string) {
	t.Helper()
	var op models.User
	if err := app.accounts(models.UserKindAdmin).Where("username = ?", seed.OperatorUsername).First(&op).Error; err != nil {
		t.Fatal(err)
	}
	reset := doJSON(t, app, http.MethodPost, "/api/v1/users/"+formatUint(op.ID)+"/reset-password", admin, nil)
	if reset.Code != http.StatusOK {
		t.Fatalf("reset: %d %s", reset.Code, reset.Body.String())
	}
	var payload struct {
		TemporaryPassword string `json:"temporaryPassword"`
	}
	if err := json.Unmarshal(decodeEnv(t, reset).Data, &payload); err != nil || payload.TemporaryPassword == "" {
		t.Fatalf("temp: %s", reset.Body.String())
	}
	tok := loginOK(t, app, seed.OperatorUsername, payload.TemporaryPassword)
	if w := doJSON(t, app, http.MethodGet, "/api/v1/auth/web-menus", tok, nil); w.Code != http.StatusOK {
		t.Fatalf("web-menus must-change: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/dicts/by/sys_gender", tok, nil); w.Code != http.StatusOK {
		t.Fatalf("dicts must-change: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/logout", tok, nil); w.Code != http.StatusOK {
		t.Fatalf("logout must-change: %d", w.Code)
	}
	disabled := doJSON(t, app, http.MethodPut, "/api/v1/users/batch-status", admin, map[string]any{
		"ids": []uint{op.ID}, "status": "disabled",
	})
	if disabled.Code != http.StatusOK {
		t.Fatalf("batch disable: %d %s", disabled.Code, disabled.Body.String())
	}
	enabled := doJSON(t, app, http.MethodPut, "/api/v1/users/batch-status", admin, map[string]any{
		"ids": []uint{op.ID}, "status": "active",
	})
	if enabled.Code != http.StatusOK {
		t.Fatalf("batch enable: %d %s", enabled.Code, enabled.Body.String())
	}
}

func coverMiscLookups(t *testing.T, app *App, admin string) {
	t.Helper()
	if w := doJSON(t, app, http.MethodGet, "/api/v1/roles?q=admin", admin, nil); w.Code != http.StatusOK {
		t.Fatalf("roles q: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/permissions?q=user", admin, nil); w.Code != http.StatusOK {
		t.Fatalf("perms q: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/forgot-password", "", map[string]string{"email": "bad"}); w.Code != http.StatusBadRequest {
		t.Fatalf("forgot bad email: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/reset-password", "", map[string]string{}); w.Code != http.StatusBadRequest {
		t.Fatalf("reset empty: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/mail/test", admin, map[string]string{"to": ""}); w.Code != http.StatusBadRequest {
		t.Fatalf("test mail empty: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/users/"+itoa(1)+"?kind=nope", admin, nil); w.Code != http.StatusBadRequest {
		t.Fatalf("bad kind query: %d", w.Code)
	}
}

