package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/seed"
	"go-react-shadcn/internal/totp"
)

func TestNavMenuCRUDAndAuthOrder(t *testing.T) {
	app := testApp(t)
	viewer := loginOK(t, app, seed.ViewerUsername, seed.ViewerPassword)
	denied := doJSON(t, app, http.MethodGet, "/api/v1/nav-menus", viewer, nil)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("viewer nav-menus: %d %s", denied.Code, denied.Body.String())
	}

	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	var dash models.NavMenu
	if err := app.DB.Where("audience = ? AND code = ?", models.NavAudienceAdmin, "dashboard:read").First(&dash).Error; err != nil {
		t.Fatal(err)
	}
	if !dash.IsSystem {
		t.Fatal("dashboard should be system")
	}
	del := doJSON(t, app, http.MethodDelete, "/api/v1/nav-menus/"+formatUint(dash.ID), admin, nil)
	if del.Code != http.StatusBadRequest || decodeEnv(t, del).ErrorCode != CodeCannotDeleteSystemMenu {
		t.Fatalf("delete system: %d %s", del.Code, del.Body.String())
	}

	if err := app.DB.Model(&dash).Updates(map[string]any{"name": "自定义仪表盘", "sort": 99, "hidden": true}).Error; err != nil {
		t.Fatal(err)
	}
	if err := seed.Run(app.DB, app.Enforcer, app.Cfg.UploadDir, true); err != nil {
		t.Fatal(err)
	}
	var again models.NavMenu
	if err := app.DB.Where("id = ?", dash.ID).First(&again).Error; err != nil {
		t.Fatal(err)
	}
	if again.Name != "自定义仪表盘" || again.Sort != 99 || !again.Hidden {
		t.Fatalf("seed overwrote admin edits: %+v", again)
	}
	_ = app.DB.Model(&again).Updates(map[string]any{"name": "仪表盘", "sort": 10, "hidden": false})

	var notify models.NavMenu
	if err := app.DB.Where("audience = ? AND code = ?", models.NavAudienceAdmin, "notify:list").First(&notify).Error; err != nil {
		t.Fatal(err)
	}
	before := doJSON(t, app, http.MethodGet, "/api/v1/auth/menus", admin, nil)
	if before.Code != http.StatusOK {
		t.Fatalf("menus: %d %s", before.Code, before.Body.String())
	}
	firstBefore := firstRootCode(t, before)
	re := doJSON(t, app, http.MethodPut, "/api/v1/nav-menus/reorder", admin, map[string]any{
		"items": []map[string]any{
			{"id": dash.ID, "sort": 50, "parentId": nil},
			{"id": notify.ID, "sort": 1, "parentId": nil},
		},
	})
	if re.Code != http.StatusOK {
		t.Fatalf("reorder: %d %s", re.Code, re.Body.String())
	}
	after := doJSON(t, app, http.MethodGet, "/api/v1/auth/menus", admin, nil)
	if after.Code != http.StatusOK {
		t.Fatalf("menus after: %d %s", after.Code, after.Body.String())
	}
	firstAfter := firstRootCode(t, after)
	if firstAfter != "notify:list" {
		t.Fatalf("expected notify first, before=%s after=%s", firstBefore, firstAfter)
	}
}

func firstRootCode(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var nodes []menuNode
	if err := json.Unmarshal(decodeEnv(t, w).Data, &nodes); err != nil || len(nodes) == 0 {
		t.Fatalf("menu tree: %v %s", err, w.Body.String())
	}
	return nodes[0].Code
}

func TestNotificationsScopedAndAnnounceByKind(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	viewerTok := loginOK(t, app, seed.ViewerUsername, seed.ViewerPassword)
	var viewer models.User
	if err := app.accounts(models.UserKindAdmin).Where("username = ?", seed.ViewerUsername).First(&viewer).Error; err != nil {
		t.Fatal(err)
	}
	app.notify(models.UserKindAdmin, viewer.ID, "roles", "角色已变更", "", "user", viewer.ID)

	listed := doJSON(t, app, http.MethodGet, "/api/v1/notifications", viewerTok, nil)
	page := decodePage[models.Notification](t, listed)
	if len(page.Items) == 0 {
		t.Fatal("viewer should see own notice")
	}
	nid := page.Items[0].ID
	stolen := doJSON(t, app, http.MethodPost, "/api/v1/notifications/"+formatUint(nid)+"/read", admin, nil)
	if stolen.Code != http.StatusNotFound {
		t.Fatalf("admin read viewer notice: %d %s", stolen.Code, stolen.Body.String())
	}
	own := doJSON(t, app, http.MethodPost, "/api/v1/notifications/"+formatUint(nid)+"/read", viewerTok, nil)
	if own.Code != http.StatusOK {
		t.Fatalf("viewer read own: %d %s", own.Code, own.Body.String())
	}

	ann := doJSON(t, app, http.MethodPost, "/api/v1/announcements", admin, map[string]string{
		"kind": "web", "title": "hello web", "body": "body",
	})
	if ann.Code != http.StatusOK {
		t.Fatalf("announce: %d %s", ann.Code, ann.Body.String())
	}
	adminBox := decodePage[models.Notification](t, doJSON(t, app, http.MethodGet, "/api/v1/notifications?unread=1", admin, nil))
	for _, n := range adminBox.Items {
		if n.Type == "announce" && n.Title == "hello web" {
			t.Fatal("admin should not receive web announcement")
		}
	}
	webTok := loginClientOK(t, app, seed.MemberUsername, seed.MemberPassword, "web")
	webBox := decodePage[models.Notification](t, doJSON(t, app, http.MethodGet, "/api/v1/notifications", webTok, nil))
	found := false
	for _, n := range webBox.Items {
		if n.Type == "announce" && n.Title == "hello web" {
			found = true
		}
	}
	if !found {
		t.Fatalf("web user missing announce: %s", doJSON(t, app, http.MethodGet, "/api/v1/notifications", webTok, nil).Body.String())
	}
}

func loginClientOK(t *testing.T, app *App, username, password, client string) string {
	t.Helper()
	id, answer, _ := issueCaptcha(t, app)
	w := doJSON(t, app, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"username": username, "password": password, "captchaId": id, "captchaCode": answer, "client": client,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("login %s: %d %s", username, w.Code, w.Body.String())
	}
	var data struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(decodeEnv(t, w).Data, &data); err != nil || data.Token == "" {
		t.Fatalf("login token: %s", w.Body.String())
	}
	return data.Token
}

func TestAdminTotpRequiredBlocksUsersUntilBound(t *testing.T) {
	app := testApp(t)
	if err := app.DB.Model(&models.SysConfig{}).Where(`"key" = ?`, "auth.admin_totp_required").Update("value", "1").Error; err != nil {
		t.Fatal(err)
	}
	app.syscfg.invalidate()

	id, answer, _ := issueCaptcha(t, app)
	w := login(t, app, seed.AdminUsername, seed.AdminPassword, id, answer)
	if w.Code != http.StatusOK {
		t.Fatalf("login: %d %s", w.Code, w.Body.String())
	}
	var step struct {
		Token        string `json:"token"`
		TotpRequired bool   `json:"totpRequired"`
		TotpEnroll   bool   `json:"totpEnroll"`
		TotpTicket   string `json:"totpTicket"`
	}
	if err := json.Unmarshal(decodeEnv(t, w).Data, &step); err != nil {
		t.Fatal(err)
	}
	if step.Token != "" || !step.TotpRequired || !step.TotpEnroll || step.TotpTicket == "" {
		t.Fatalf("expected enroll ticket, got %+v %s", step, w.Body.String())
	}
	blocked := doJSON(t, app, http.MethodGet, "/api/v1/users", step.TotpTicket, nil)
	if blocked.Code != http.StatusUnauthorized {
		t.Fatalf("ticket as bearer: %d %s", blocked.Code, blocked.Body.String())
	}

	setup := doJSON(t, app, http.MethodPost, "/api/v1/auth/totp/setup", "", map[string]string{"ticket": step.TotpTicket})
	if setup.Code != http.StatusOK {
		t.Fatalf("setup: %d %s", setup.Code, setup.Body.String())
	}
	var enroll struct {
		Secret string `json:"secret"`
		Ticket string `json:"ticket"`
	}
	if err := json.Unmarshal(decodeEnv(t, setup).Data, &enroll); err != nil || enroll.Secret == "" {
		t.Fatalf("setup data: %s", setup.Body.String())
	}
	code, err := totp.Code(enroll.Secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	confirm := doJSON(t, app, http.MethodPost, "/api/v1/auth/totp/confirm", "", map[string]string{
		"ticket": enroll.Ticket, "code": code,
	})
	if confirm.Code != http.StatusOK {
		t.Fatalf("confirm: %d %s", confirm.Code, confirm.Body.String())
	}
	var done struct {
		Token         string   `json:"token"`
		User          userDTO  `json:"user"`
		RecoveryCodes []string `json:"recoveryCodes"`
	}
	if err := json.Unmarshal(decodeEnv(t, confirm).Data, &done); err != nil || done.Token == "" {
		t.Fatalf("confirm data: %s", confirm.Body.String())
	}
	if len(done.RecoveryCodes) == 0 {
		t.Fatal("expected recovery codes")
	}
	users := doJSON(t, app, http.MethodGet, "/api/v1/users", done.Token, nil)
	if users.Code != http.StatusOK {
		t.Fatalf("users after totp: %d %s", users.Code, users.Body.String())
	}
	body := users.Body.String()
	if strings.Contains(body, enroll.Secret) || strings.Contains(body, `"totpSecret"`) || strings.Contains(body, "totp_secret") {
		t.Fatalf("secret leaked in users list: %s", body)
	}
	me := doJSON(t, app, http.MethodGet, "/api/v1/auth/me", done.Token, nil)
	if me.Code != http.StatusOK {
		t.Fatalf("me: %d %s", me.Code, me.Body.String())
	}
	if strings.Contains(me.Body.String(), enroll.Secret) || strings.Contains(me.Body.String(), `"totpSecret"`) {
		t.Fatalf("secret leaked in me: %s", me.Body.String())
	}
	if !strings.Contains(me.Body.String(), `"totpEnabled":true`) {
		t.Fatalf("me missing totpEnabled: %s", me.Body.String())
	}
}
