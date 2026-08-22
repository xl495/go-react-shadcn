package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"go-react-shadcn/internal/googleid"
	"go-react-shadcn/internal/mailer"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/seed"
)

func TestOwnSessionsCannotKickOthers(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	viewer := loginOK(t, app, seed.ViewerUsername, seed.ViewerPassword)

	own := doJSON(t, app, http.MethodGet, "/api/v1/auth/sessions", viewer, nil)
	if own.Code != http.StatusOK {
		t.Fatalf("own sessions: %d %s", own.Code, own.Body.String())
	}
	var mine []models.AuthSession
	if err := json.Unmarshal(decodeEnv(t, own).Data, &mine); err != nil || len(mine) == 0 {
		t.Fatalf("viewer sessions: %v %s", err, own.Body.String())
	}

	adminList := doJSON(t, app, http.MethodGet, "/api/v1/auth/sessions", admin, nil)
	var admins []models.AuthSession
	if err := json.Unmarshal(decodeEnv(t, adminList).Data, &admins); err != nil || len(admins) == 0 {
		t.Fatalf("admin sessions: %v %s", err, adminList.Body.String())
	}
	stolen := doJSON(t, app, http.MethodDelete, "/api/v1/auth/sessions/"+formatUint(admins[0].ID), viewer, nil)
	if stolen.Code != http.StatusNotFound || decodeEnv(t, stolen).ErrorCode != CodeSessionNotFound {
		t.Fatalf("viewer kick admin session: %d %s", stolen.Code, stolen.Body.String())
	}
	denied := doJSON(t, app, http.MethodGet, "/api/v1/users/"+formatUint(admins[0].UserID)+"/sessions", viewer, nil)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("viewer list admin sessions: %d %s", denied.Code, denied.Body.String())
	}
}

func TestMustChangePasswordBlocksUsers(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	var op models.User
	if err := app.accounts(models.UserKindAdmin).Where("username = ?", seed.OperatorUsername).First(&op).Error; err != nil {
		t.Fatal(err)
	}
	reset := doJSON(t, app, http.MethodPost, "/api/v1/users/"+formatUint(op.ID)+"/reset-password", admin, nil)
	if reset.Code != http.StatusOK {
		t.Fatalf("reset: %d %s", reset.Code, reset.Body.String())
	}
	var payload struct {
		TemporaryPassword  string `json:"temporaryPassword"`
		MustChangePassword bool   `json:"mustChangePassword"`
	}
	if err := json.Unmarshal(decodeEnv(t, reset).Data, &payload); err != nil || payload.TemporaryPassword == "" || !payload.MustChangePassword {
		t.Fatalf("reset payload: %s", reset.Body.String())
	}
	tok := loginOK(t, app, seed.OperatorUsername, payload.TemporaryPassword)
	blocked := doJSON(t, app, http.MethodGet, "/api/v1/users", tok, nil)
	if blocked.Code != http.StatusForbidden || decodeEnv(t, blocked).ErrorCode != CodeMustChangePassword {
		t.Fatalf("must change users: %d %s", blocked.Code, blocked.Body.String())
	}
	me := doJSON(t, app, http.MethodGet, "/api/v1/auth/me", tok, nil)
	if me.Code != http.StatusOK {
		t.Fatalf("me: %d %s", me.Code, me.Body.String())
	}
	changed := doJSON(t, app, http.MethodPut, "/api/v1/auth/password", tok, map[string]string{
		"oldPassword": payload.TemporaryPassword, "newPassword": "operator-pass-9",
	})
	if changed.Code != http.StatusOK {
		t.Fatalf("change: %d %s", changed.Code, changed.Body.String())
	}
	fresh := loginOK(t, app, seed.OperatorUsername, "operator-pass-9")
	users := doJSON(t, app, http.MethodGet, "/api/v1/users", fresh, nil)
	if users.Code != http.StatusOK {
		t.Fatalf("users after change: %d %s", users.Code, users.Body.String())
	}
}

func TestUnlockAndCopyRoleAndBatchStatus(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	viewer := loginOK(t, app, seed.ViewerUsername, seed.ViewerPassword)

	var op models.User
	if err := app.accounts(models.UserKindAdmin).Where("username = ?", seed.OperatorUsername).First(&op).Error; err != nil {
		t.Fatal(err)
	}
	until := time.Now().Add(time.Hour)
	if err := app.accounts(models.UserKindAdmin).Where("id = ?", op.ID).Updates(map[string]any{
		"locked_until": until, "failed_login_count": 9,
	}).Error; err != nil {
		t.Fatal(err)
	}
	detail := doJSON(t, app, http.MethodGet, "/api/v1/users/"+formatUint(op.ID), admin, nil)
	if detail.Code != http.StatusOK || !strings.Contains(detail.Body.String(), "lockedUntil") {
		t.Fatalf("lockedUntil missing: %s", detail.Body.String())
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/users/"+formatUint(op.ID)+"/unlock", viewer, nil); w.Code != http.StatusForbidden {
		t.Fatalf("viewer unlock: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/users/"+formatUint(op.ID)+"/unlock", admin, nil); w.Code != http.StatusOK {
		t.Fatalf("admin unlock: %d %s", w.Code, w.Body.String())
	}
	if tok := loginOK(t, app, seed.OperatorUsername, seed.OperatorPassword); tok == "" {
		t.Fatal("login after unlock")
	}

	var viewerRole models.Role
	if err := app.DB.Where("code = ?", seed.RoleViewer).First(&viewerRole).Error; err != nil {
		t.Fatal(err)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/roles/"+formatUint(viewerRole.ID)+"/copy", viewer, map[string]string{
		"name": "cloned", "code": "cloned-viewer",
	}); w.Code != http.StatusForbidden {
		t.Fatalf("viewer copy: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/roles/"+formatUint(viewerRole.ID)+"/copy", admin, map[string]string{
		"name": "cloned", "code": seed.RoleAdmin,
	}); w.Code != http.StatusBadRequest {
		t.Fatalf("copy builtin: %d %s", w.Code, w.Body.String())
	}
	copied := doJSON(t, app, http.MethodPost, "/api/v1/roles/"+formatUint(viewerRole.ID)+"/copy", admin, map[string]string{
		"name": "观察员副本", "code": "viewer-copy",
	})
	if copied.Code != http.StatusOK {
		t.Fatalf("copy: %d %s", copied.Code, copied.Body.String())
	}

	var adminUser models.User
	if err := app.accounts(models.UserKindAdmin).Where("username = ?", seed.AdminUsername).First(&adminUser).Error; err != nil {
		t.Fatal(err)
	}
	last := doJSON(t, app, http.MethodPut, "/api/v1/users/batch-status", admin, map[string]any{
		"ids": []uint{adminUser.ID}, "status": "disabled", "kind": "admin",
	})
	if last.Code != http.StatusOK {
		t.Fatalf("disable last admin: %d %s", last.Code, last.Body.String())
	}
	var lastBatch struct {
		Updated []uint `json:"updated"`
		Skipped []uint `json:"skipped"`
	}
	if err := json.Unmarshal(decodeEnv(t, last).Data, &lastBatch); err != nil {
		t.Fatal(err)
	}
	if len(lastBatch.Updated) != 0 || len(lastBatch.Skipped) != 1 || lastBatch.Skipped[0] != adminUser.ID {
		t.Fatalf("last admin should be skipped: %+v", lastBatch)
	}
	if w := doJSON(t, app, http.MethodPut, "/api/v1/users/batch-status", viewer, map[string]any{
		"ids": []uint{op.ID}, "status": "disabled", "kind": "admin",
	}); w.Code != http.StatusForbidden {
		t.Fatalf("viewer batch: %d %s", w.Code, w.Body.String())
	}
	disabled := doJSON(t, app, http.MethodPut, "/api/v1/users/batch-status", admin, map[string]any{
		"ids": []uint{op.ID}, "status": "disabled", "kind": "admin",
	})
	if disabled.Code != http.StatusOK {
		t.Fatalf("batch disable: %d %s", disabled.Code, disabled.Body.String())
	}
}

func TestBatchStatusHonorsDataScope(t *testing.T) {
	app := testApp(t)
	var opRole models.Role
	if err := app.DB.Preload("Permissions").Where("code = ?", seed.RoleOperator).First(&opRole).Error; err != nil {
		t.Fatal(err)
	}
	var update models.Permission
	if err := app.DB.Where("code = ?", "user:update").First(&update).Error; err != nil {
		t.Fatal(err)
	}
	perms := opRole.Permissions
	perms = append(perms, update)
	if err := app.DB.Model(&opRole).Association("Permissions").Replace(perms); err != nil {
		t.Fatal(err)
	}
	if err := seed.SyncRolePolicies(app.Enforcer, opRole.Code, perms); err != nil {
		t.Fatal(err)
	}
	var adminUser, opUser models.User
	if err := app.accounts(models.UserKindAdmin).Where("username = ?", seed.AdminUsername).First(&adminUser).Error; err != nil {
		t.Fatal(err)
	}
	if err := app.accounts(models.UserKindAdmin).Where("username = ?", seed.OperatorUsername).First(&opUser).Error; err != nil {
		t.Fatal(err)
	}
	opTok := loginOK(t, app, seed.OperatorUsername, seed.OperatorPassword)
	w := doJSON(t, app, http.MethodPut, "/api/v1/users/batch-status", opTok, map[string]any{
		"ids": []uint{adminUser.ID, opUser.ID}, "status": "disabled", "kind": "admin",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("scoped batch: %d %s", w.Code, w.Body.String())
	}
	var batch struct {
		Updated []uint `json:"updated"`
		Skipped []uint `json:"skipped"`
	}
	if err := json.Unmarshal(decodeEnv(t, w).Data, &batch); err != nil {
		t.Fatal(err)
	}
	foundSkip := false
	for _, id := range batch.Skipped {
		if id == adminUser.ID {
			foundSkip = true
			break
		}
	}
	if !foundSkip {
		t.Fatalf("admin id should be skipped: %+v", batch)
	}
	var again models.User
	if err := app.accounts(models.UserKindAdmin).Where("id = ?", adminUser.ID).First(&again).Error; err != nil {
		t.Fatal(err)
	}
	if again.Status != "active" {
		t.Fatalf("admin should stay active, got %s", again.Status)
	}
}

func TestMaintenanceBlocksWebNotAdmin(t *testing.T) {
	app := testApp(t)
	setCfg(t, app, "app.maintenance", "1")
	settings := doJSON(t, app, http.MethodGet, "/api/v1/auth/settings", "", nil)
	if settings.Code != http.StatusOK || !strings.Contains(settings.Body.String(), `"maintenance":true`) {
		t.Fatalf("settings: %s", settings.Body.String())
	}
	id, ans, _ := issueCaptcha(t, app)
	web := doJSON(t, app, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"username": seed.MemberUsername, "password": seed.MemberPassword, "client": "web",
		"captchaId": id, "captchaCode": ans,
	})
	if web.Code != http.StatusServiceUnavailable || decodeEnv(t, web).ErrorCode != CodeMaintenance {
		t.Fatalf("web login: %d %s", web.Code, web.Body.String())
	}
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	if admin == "" {
		t.Fatal("admin login during maintenance")
	}
	live := doJSON(t, app, http.MethodGet, "/live", "", nil)
	if live.Code != http.StatusOK {
		t.Fatalf("live: %d", live.Code)
	}
}

func TestGoogleBindUnbindAndNoSecretLeak(t *testing.T) {
	app := testApp(t)
	setCfg(t, app, "auth.google_enabled", "1")
	setCfg(t, app, "auth.google_client_id", "client-1")
	app.GoogleVerify = stubGoogle{ident: googleid.Identity{
		Subject: "gid-viewer", Email: "viewer@latch.local", EmailVerified: true, Name: "李访客",
	}}
	viewer := loginOK(t, app, seed.ViewerUsername, seed.ViewerPassword)
	off := doJSON(t, app, http.MethodPost, "/api/v1/auth/google/bind", viewer, map[string]string{"idToken": "tok"})
	if off.Code != http.StatusOK {
		t.Fatalf("bind with google on: %d %s", off.Code, off.Body.String())
	}
	me := doJSON(t, app, http.MethodGet, "/api/v1/auth/me", viewer, nil)
	if me.Code != http.StatusOK {
		t.Fatalf("me: %d %s", me.Code, me.Body.String())
	}
	body := me.Body.String()
	if strings.Contains(body, "gid-viewer") || strings.Contains(body, `"google_id"`) || strings.Contains(body, `"totpSecret"`) {
		t.Fatalf("secret leak: %s", body)
	}
	if !strings.Contains(body, `"googleBound":true`) {
		t.Fatalf("missing googleBound: %s", body)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/auth/google/unbind", viewer, nil); w.Code != http.StatusBadRequest {
		t.Fatalf("unbind without password: %d %s", w.Code, w.Body.String())
	}
	un := doJSON(t, app, http.MethodPost, "/api/v1/auth/google/unbind", viewer, map[string]string{
		"password": seed.ViewerPassword,
	})
	if un.Code != http.StatusOK {
		t.Fatalf("unbind: %d %s", un.Code, un.Body.String())
	}
	if tok := loginOK(t, app, seed.ViewerUsername, seed.ViewerPassword); tok == "" {
		t.Fatal("password login after unbind")
	}
	setCfg(t, app, "auth.google_enabled", "0")
	again := doJSON(t, app, http.MethodPost, "/api/v1/auth/google/bind", viewer, map[string]string{"idToken": "tok"})
	if again.Code != http.StatusServiceUnavailable {
		t.Fatalf("bind while off: %d %s", again.Code, again.Body.String())
	}
}

func TestOwnLoginLogsAndOnlineSessions(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	_ = loginOK(t, app, seed.ViewerUsername, seed.ViewerPassword)
	viewer := loginOK(t, app, seed.ViewerUsername, seed.ViewerPassword)

	logs := doJSON(t, app, http.MethodGet, "/api/v1/auth/login-logs", admin, nil)
	page := decodePage[models.LoginLog](t, logs)
	for _, row := range page.Items {
		if row.Username != seed.AdminUsername {
			t.Fatalf("admin login-logs leaked %q", row.Username)
		}
	}
	vlogs := doJSON(t, app, http.MethodGet, "/api/v1/auth/login-logs?username="+seed.AdminUsername, viewer, nil)
	vpage := decodePage[models.LoginLog](t, vlogs)
	for _, row := range vpage.Items {
		if row.Username != seed.ViewerUsername {
			t.Fatalf("viewer saw %q", row.Username)
		}
	}

	if w := doJSON(t, app, http.MethodGet, "/api/v1/online-sessions", viewer, nil); w.Code != http.StatusForbidden {
		t.Fatalf("viewer online: %d %s", w.Code, w.Body.String())
	}
	online := doJSON(t, app, http.MethodGet, "/api/v1/online-sessions", admin, nil)
	if online.Code != http.StatusOK || !strings.Contains(online.Body.String(), seed.AdminUsername) {
		t.Fatalf("online: %d %s", online.Code, online.Body.String())
	}
}

func TestEmailChangeRequiresVerify(t *testing.T) {
	app := testApp(t)
	viewer := loginOK(t, app, seed.ViewerUsername, seed.ViewerPassword)
	updated := doJSON(t, app, http.MethodPut, "/api/v1/auth/profile", viewer, map[string]any{
		"nickname": "李访客", "email": "viewer2@example.com", "phone": "13800000003",
		"gender": "female", "department": "market", "title": "观察员", "remark": "",
		"timezone": "Asia/Shanghai",
	})
	if updated.Code != http.StatusOK {
		t.Fatalf("profile: %d %s", updated.Code, updated.Body.String())
	}
	var dto userDTO
	if err := json.Unmarshal(decodeEnv(t, updated).Data, &dto); err != nil {
		t.Fatal(err)
	}
	if dto.Email != "viewer@latch.local" || dto.PendingEmail != "viewer2@example.com" || dto.EmailVerifyToken == "" {
		t.Fatalf("pending not applied: %+v", dto)
	}
	me := doJSON(t, app, http.MethodGet, "/api/v1/auth/me", viewer, nil)
	var meDTO userDTO
	if err := json.Unmarshal(decodeEnv(t, me).Data, &meDTO); err != nil {
		t.Fatal(err)
	}
	if meDTO.Email != "viewer@latch.local" {
		t.Fatalf("me email changed early: %s", me.Body.String())
	}
	taken := doJSON(t, app, http.MethodPut, "/api/v1/auth/profile", viewer, map[string]any{
		"nickname": "李访客", "email": "admin@latch.local", "phone": "13800000003",
		"gender": "female", "department": "market", "title": "观察员", "remark": "",
	})
	if taken.Code != http.StatusConflict {
		t.Fatalf("taken email: %d %s", taken.Code, taken.Body.String())
	}
	opTok := loginOK(t, app, seed.OperatorUsername, seed.OperatorPassword)
	pendingClash := doJSON(t, app, http.MethodPut, "/api/v1/auth/profile", opTok, map[string]any{
		"nickname": "张操作", "email": "viewer2@example.com", "phone": "13800000002",
		"gender": "male", "department": "ops", "title": "操作员", "remark": "",
	})
	if pendingClash.Code != http.StatusConflict {
		t.Fatalf("pending email clash: %d %s", pendingClash.Code, pendingClash.Body.String())
	}
	confirm := doJSON(t, app, http.MethodPost, "/api/v1/auth/verify-email", "", map[string]string{"token": dto.EmailVerifyToken})
	if confirm.Code != http.StatusOK {
		t.Fatalf("verify: %d %s", confirm.Code, confirm.Body.String())
	}
	var changed struct {
		Changed bool `json:"changed"`
	}
	if err := json.Unmarshal(decodeEnv(t, confirm).Data, &changed); err != nil || !changed.Changed {
		t.Fatalf("email-change verify contract: %s", confirm.Body.String())
	}
	me2 := doJSON(t, app, http.MethodGet, "/api/v1/auth/me", viewer, nil)
	if !strings.Contains(me2.Body.String(), "viewer2@example.com") {
		t.Fatalf("email not updated: %s", me2.Body.String())
	}
	var jobs int64
	if err := app.DB.Model(&models.MailJob{}).Where("to_email = ?", "viewer2@example.com").Count(&jobs).Error; err != nil || jobs < 1 {
		t.Fatalf("email-change mail not queued: jobs=%d err=%v", jobs, err)
	}
}

func TestLoginLogsStayOnOwnKind(t *testing.T) {
	app := testApp(t)
	if err := app.DB.Create(&models.LoginLog{
		Username: seed.AdminUsername, UserKind: models.UserKindWeb, Status: "success", IP: "9.9.9.9",
	}).Error; err != nil {
		t.Fatal(err)
	}
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	logs := doJSON(t, app, http.MethodGet, "/api/v1/auth/login-logs", admin, nil)
	page := decodePage[models.LoginLog](t, logs)
	for _, row := range page.Items {
		if row.IP == "9.9.9.9" || row.UserKind == models.UserKindWeb {
			t.Fatalf("admin saw web login log: %+v", row)
		}
	}
}

func TestLockRevokesExistingJWT(t *testing.T) {
	app := testApp(t)
	tok := loginOK(t, app, seed.OperatorUsername, seed.OperatorPassword)
	for i := 0; i < app.LoginGuard.MaxFailures(); i++ {
		id, ans, _ := issueCaptcha(t, app)
		w := login(t, app, seed.OperatorUsername, "wrong-password-9", id, ans)
		if w.Code == http.StatusOK {
			t.Fatalf("wrong password succeeded on attempt %d", i)
		}
	}
	me := doJSON(t, app, http.MethodGet, "/api/v1/auth/me", tok, nil)
	code := decodeEnv(t, me).ErrorCode
	if me.Code == http.StatusOK || (code != CodeAccountLocked && code != CodeInvalidToken) {
		t.Fatalf("locked jwt still valid: %d %s", me.Code, me.Body.String())
	}
	var op models.User
	if err := app.accounts(models.UserKindAdmin).Where("username = ?", seed.OperatorUsername).First(&op).Error; err != nil {
		t.Fatal(err)
	}
	until := time.Now().Add(time.Hour)
	if err := app.updateAccount(&op, map[string]any{"locked_until": until, "failed_login_count": 9}); err != nil {
		t.Fatal(err)
	}
	fresh := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	_ = doJSON(t, app, http.MethodPost, "/api/v1/users/"+formatUint(op.ID)+"/unlock", fresh, nil)
	tok2 := loginOK(t, app, seed.OperatorUsername, seed.OperatorPassword)
	if err := app.updateAccount(&op, map[string]any{"locked_until": until}); err != nil {
		t.Fatal(err)
	}
	app.sessions.invalidate(models.UserKindAdmin, op.ID)
	locked := doJSON(t, app, http.MethodGet, "/api/v1/auth/me", tok2, nil)
	if locked.Code != http.StatusForbidden || decodeEnv(t, locked).ErrorCode != CodeAccountLocked {
		t.Fatalf("lock bit ignored: %d %s", locked.Code, locked.Body.String())
	}
}

func TestLastAdminCannotBeRemoved(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	var user models.User
	if err := app.accounts(models.UserKindAdmin).Where("username = ?", seed.AdminUsername).First(&user).Error; err != nil {
		t.Fatal(err)
	}
	disabled := doJSON(t, app, http.MethodPut, "/api/v1/users/"+formatUint(user.ID), admin, map[string]any{
		"status": "disabled",
	})
	if disabled.Code != http.StatusBadRequest || decodeEnv(t, disabled).ErrorCode != CodeCannotDisableLastAdmin {
		t.Fatalf("disable last admin: %d %s", disabled.Code, disabled.Body.String())
	}
	var viewerRole models.Role
	if err := app.DB.Where("code = ?", seed.RoleViewer).First(&viewerRole).Error; err != nil {
		t.Fatal(err)
	}
	demote := doJSON(t, app, http.MethodPut, "/api/v1/users/"+formatUint(user.ID)+"/roles", admin, map[string]any{
		"roleIds": []uint{viewerRole.ID},
	})
	if demote.Code != http.StatusBadRequest || decodeEnv(t, demote).ErrorCode != CodeCannotDisableLastAdmin {
		t.Fatalf("demote last admin: %d %s", demote.Code, demote.Body.String())
	}
}

func TestOwnSessionsMarkCurrent(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	w := doJSON(t, app, http.MethodGet, "/api/v1/auth/sessions", admin, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("sessions: %d %s", w.Code, w.Body.String())
	}
	var rows []ownSessionDTO
	if err := json.Unmarshal(decodeEnv(t, w).Data, &rows); err != nil || len(rows) == 0 {
		t.Fatalf("decode sessions: %v %s", err, w.Body.String())
	}
	current := 0
	for _, row := range rows {
		if row.Current {
			current++
		}
	}
	if current != 1 {
		t.Fatalf("current sessions=%d want 1 body=%s", current, w.Body.String())
	}
}

func TestMaintenanceBlocksWebVerifyAndReset(t *testing.T) {
	app := testApp(t)
	var webUser, adminUser models.User
	if err := app.accounts(models.UserKindWeb).Where("username = ?", seed.MemberUsername).First(&webUser).Error; err != nil {
		t.Fatal(err)
	}
	if err := app.accounts(models.UserKindAdmin).Where("username = ?", seed.ViewerUsername).First(&adminUser).Error; err != nil {
		t.Fatal(err)
	}
	rawVerify, hashVerify, err := mailer.NewResetToken()
	if err != nil {
		t.Fatal(err)
	}
	rawReset, hashReset, err := mailer.NewResetToken()
	if err != nil {
		t.Fatal(err)
	}
	rawAdmin, hashAdmin, err := mailer.NewResetToken()
	if err != nil {
		t.Fatal(err)
	}
	exp := time.Now().Add(time.Hour)
	if err := app.DB.Create(&models.PasswordResetToken{
		UserID: webUser.ID, UserKind: models.UserKindWeb, Purpose: models.TokenPurposeVerify,
		TokenHash: hashVerify, ExpiresAt: exp,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := app.DB.Create(&models.PasswordResetToken{
		UserID: webUser.ID, UserKind: models.UserKindWeb, Purpose: models.TokenPurposeReset,
		TokenHash: hashReset, ExpiresAt: exp,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := app.DB.Create(&models.PasswordResetToken{
		UserID: adminUser.ID, UserKind: models.UserKindAdmin, Purpose: models.TokenPurposeVerify,
		TokenHash: hashAdmin, ExpiresAt: exp,
	}).Error; err != nil {
		t.Fatal(err)
	}

	setCfg(t, app, "app.maintenance", "1")
	id, ans, _ := issueCaptcha(t, app)
	forgot := doJSON(t, app, http.MethodPost, "/api/v1/auth/forgot-password", "", map[string]string{
		"email": "webuser@latch.local", "client": "web", "captchaId": id, "captchaCode": ans,
	})
	if forgot.Code != http.StatusServiceUnavailable || decodeEnv(t, forgot).ErrorCode != CodeMaintenance {
		t.Fatalf("web forgot: %d %s", forgot.Code, forgot.Body.String())
	}
	verify := doJSON(t, app, http.MethodPost, "/api/v1/auth/verify-email", "", map[string]string{"token": rawVerify})
	if verify.Code != http.StatusServiceUnavailable || decodeEnv(t, verify).ErrorCode != CodeMaintenance {
		t.Fatalf("web verify: %d %s", verify.Code, verify.Body.String())
	}
	reset := doJSON(t, app, http.MethodPost, "/api/v1/auth/reset-password", "", map[string]string{
		"token": rawReset, "newPassword": "webuser-new-99",
	})
	if reset.Code != http.StatusServiceUnavailable || decodeEnv(t, reset).ErrorCode != CodeMaintenance {
		t.Fatalf("web reset: %d %s", reset.Code, reset.Body.String())
	}
	adminVerify := doJSON(t, app, http.MethodPost, "/api/v1/auth/verify-email", "", map[string]string{"token": rawAdmin})
	if adminVerify.Code != http.StatusOK {
		t.Fatalf("admin verify during maintenance: %d %s", adminVerify.Code, adminVerify.Body.String())
	}
}

func TestCannotRevokeCurrentSession(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	w := doJSON(t, app, http.MethodGet, "/api/v1/auth/sessions", admin, nil)
	var rows []ownSessionDTO
	if err := json.Unmarshal(decodeEnv(t, w).Data, &rows); err != nil || len(rows) == 0 {
		t.Fatalf("sessions: %v %s", err, w.Body.String())
	}
	var current uint
	for _, row := range rows {
		if row.Current {
			current = row.ID
			break
		}
	}
	if current == 0 {
		t.Fatal("missing current session")
	}
	denied := doJSON(t, app, http.MethodDelete, "/api/v1/auth/sessions/"+formatUint(current), admin, nil)
	if denied.Code != http.StatusBadRequest || decodeEnv(t, denied).ErrorCode != CodeCannotRevokeCurrent {
		t.Fatalf("revoke current: %d %s", denied.Code, denied.Body.String())
	}
}

func TestForgotPasswordKeepsEmailChangeToken(t *testing.T) {
	app := testApp(t)
	var viewer models.User
	if err := app.accounts(models.UserKindAdmin).Where("username = ?", seed.ViewerUsername).First(&viewer).Error; err != nil {
		t.Fatal(err)
	}
	_, hash, err := mailer.NewResetToken()
	if err != nil {
		t.Fatal(err)
	}
	if err := app.DB.Create(&models.PasswordResetToken{
		UserID: viewer.ID, UserKind: models.UserKindAdmin, Purpose: models.TokenPurposeEmailChange,
		TokenHash: hash, ExpiresAt: time.Now().Add(time.Hour),
	}).Error; err != nil {
		t.Fatal(err)
	}
	id, ans, _ := issueCaptcha(t, app)
	forgot := doJSON(t, app, http.MethodPost, "/api/v1/auth/forgot-password", "", map[string]string{
		"email": "viewer@latch.local", "captchaId": id, "captchaCode": ans,
	})
	if forgot.Code != http.StatusOK {
		t.Fatalf("forgot: %d %s", forgot.Code, forgot.Body.String())
	}
	var n int64
	if err := app.DB.Model(&models.PasswordResetToken{}).
		Where("user_id = ? AND user_kind = ? AND purpose = ? AND used_at IS NULL", viewer.ID, models.UserKindAdmin, models.TokenPurposeEmailChange).
		Count(&n).Error; err != nil || n != 1 {
		t.Fatalf("email-change token wiped: n=%d err=%v", n, err)
	}
}

func TestResetPasswordClearsLockAndMustChange(t *testing.T) {
	app := testApp(t)
	var op models.User
	if err := app.accounts(models.UserKindAdmin).Where("username = ?", seed.OperatorUsername).First(&op).Error; err != nil {
		t.Fatal(err)
	}
	until := time.Now().Add(time.Hour)
	if err := app.updateAccount(&op, map[string]any{
		"locked_until": until, "failed_login_count": 9, "must_change_password": true,
	}); err != nil {
		t.Fatal(err)
	}
	raw, hash, err := mailer.NewResetToken()
	if err != nil {
		t.Fatal(err)
	}
	if err := app.DB.Create(&models.PasswordResetToken{
		UserID: op.ID, UserKind: models.UserKindAdmin, Purpose: models.TokenPurposeReset,
		TokenHash: hash, ExpiresAt: time.Now().Add(time.Hour),
	}).Error; err != nil {
		t.Fatal(err)
	}
	reset := doJSON(t, app, http.MethodPost, "/api/v1/auth/reset-password", "", map[string]string{
		"token": raw, "newPassword": "operator-reset-9",
	})
	if reset.Code != http.StatusOK {
		t.Fatalf("reset: %d %s", reset.Code, reset.Body.String())
	}
	tok := loginOK(t, app, seed.OperatorUsername, "operator-reset-9")
	dicts := doJSON(t, app, http.MethodGet, "/api/v1/dicts", tok, nil)
	if dicts.Code != http.StatusOK {
		t.Fatalf("must-change after self reset: %d %s", dicts.Code, dicts.Body.String())
	}
}

func TestAdminUpdateUserClearsPendingEmail(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	var viewer models.User
	if err := app.accounts(models.UserKindAdmin).Where("username = ?", seed.ViewerUsername).First(&viewer).Error; err != nil {
		t.Fatal(err)
	}
	_, hash, err := mailer.NewResetToken()
	if err != nil {
		t.Fatal(err)
	}
	if err := app.updateAccount(&viewer, map[string]any{"pending_email": "viewer-next@example.com"}); err != nil {
		t.Fatal(err)
	}
	if err := app.DB.Create(&models.PasswordResetToken{
		UserID: viewer.ID, UserKind: models.UserKindAdmin, Purpose: models.TokenPurposeEmailChange,
		TokenHash: hash, ExpiresAt: time.Now().Add(time.Hour),
	}).Error; err != nil {
		t.Fatal(err)
	}
	updated := doJSON(t, app, http.MethodPut, "/api/v1/users/"+formatUint(viewer.ID), admin, map[string]any{
		"email": "viewer-admin@example.com",
	})
	if updated.Code != http.StatusOK {
		t.Fatalf("update: %d %s", updated.Code, updated.Body.String())
	}
	var dto userDTO
	if err := json.Unmarshal(decodeEnv(t, updated).Data, &dto); err != nil {
		t.Fatal(err)
	}
	if dto.Email != "viewer-admin@example.com" || dto.PendingEmail != "" || !dto.EmailVerified {
		t.Fatalf("dto: %+v", dto)
	}
	var n int64
	if err := app.DB.Model(&models.PasswordResetToken{}).
		Where("user_id = ? AND user_kind = ? AND purpose = ? AND used_at IS NULL", viewer.ID, models.UserKindAdmin, models.TokenPurposeEmailChange).
		Count(&n).Error; err != nil || n != 0 {
		t.Fatalf("email-change token remains n=%d err=%v", n, err)
	}
}

func TestGoogleBindRejectsTakenEmail(t *testing.T) {
	app := testApp(t)
	setCfg(t, app, "auth.google_enabled", "1")
	setCfg(t, app, "auth.google_client_id", "client-1")
	var viewer models.User
	if err := app.accounts(models.UserKindAdmin).Where("username = ?", seed.ViewerUsername).First(&viewer).Error; err != nil {
		t.Fatal(err)
	}
	if err := app.updateAccount(&viewer, map[string]any{"email": ""}); err != nil {
		t.Fatal(err)
	}
	app.GoogleVerify = stubGoogle{ident: googleid.Identity{
		Subject: "gid-taken-mail", Email: "admin@latch.local", EmailVerified: true,
	}}
	tok := loginOK(t, app, seed.ViewerUsername, seed.ViewerPassword)
	denied := doJSON(t, app, http.MethodPost, "/api/v1/auth/google/bind", tok, map[string]string{"idToken": "tok"})
	if denied.Code != http.StatusConflict || decodeEnv(t, denied).ErrorCode != CodeEmailExists {
		t.Fatalf("bind taken email: %d %s", denied.Code, denied.Body.String())
	}
}

func TestGoogleLoginSkipsTakenEmptyEmailFill(t *testing.T) {
	app := testApp(t)
	setCfg(t, app, "auth.google_enabled", "1")
	setCfg(t, app, "auth.google_client_id", "client-1")
	hash, err := randomUnusableHash()
	if err != nil {
		t.Fatal(err)
	}
	user := models.User{
		Username: "gempty", PasswordHash: hash, Status: "active",
		Timezone: mailer.DefaultTimezone, Kind: models.UserKindWeb, GoogleID: "gid-empty",
	}
	if err := app.accounts(models.UserKindWeb).Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	app.GoogleVerify = stubGoogle{ident: googleid.Identity{
		Subject: "gid-empty", Email: "webuser@latch.local", EmailVerified: true,
	}}
	w := doJSON(t, app, http.MethodPost, "/api/v1/auth/google", "", map[string]string{
		"idToken": "tok", "client": "web",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("google: %d %s", w.Code, w.Body.String())
	}
	var row models.User
	if err := app.accounts(models.UserKindWeb).Where("username = ?", "gempty").First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Email != "" {
		t.Fatalf("filled taken email: %q", row.Email)
	}
}

func TestCancelPendingEmailChange(t *testing.T) {
	app := testApp(t)
	viewer := loginOK(t, app, seed.ViewerUsername, seed.ViewerPassword)
	updated := doJSON(t, app, http.MethodPut, "/api/v1/auth/profile", viewer, map[string]any{
		"nickname": "李访客", "email": "viewer-cancel@example.com", "phone": "13800000003",
		"gender": "female", "department": "market", "title": "观察员", "remark": "",
		"timezone": "Asia/Shanghai",
	})
	if updated.Code != http.StatusOK {
		t.Fatalf("profile: %d %s", updated.Code, updated.Body.String())
	}
	var dto userDTO
	if err := json.Unmarshal(decodeEnv(t, updated).Data, &dto); err != nil || dto.PendingEmail == "" {
		t.Fatalf("pending: %s", updated.Body.String())
	}
	canceled := doJSON(t, app, http.MethodDelete, "/api/v1/auth/pending-email", viewer, nil)
	if canceled.Code != http.StatusOK {
		t.Fatalf("cancel: %d %s", canceled.Code, canceled.Body.String())
	}
	var after userDTO
	if err := json.Unmarshal(decodeEnv(t, canceled).Data, &after); err != nil || after.PendingEmail != "" || after.Email != "viewer@latch.local" {
		t.Fatalf("after cancel: %+v %s", after, canceled.Body.String())
	}
	var row models.User
	if err := app.accounts(models.UserKindAdmin).Where("username = ?", seed.ViewerUsername).First(&row).Error; err != nil || row.PendingEmail != "" {
		t.Fatalf("db pending=%q err=%v", row.PendingEmail, err)
	}
}

func TestGoogleUserCanSetPasswordWithoutOld(t *testing.T) {
	app := testApp(t)
	setCfg(t, app, "auth.google_enabled", "1")
	setCfg(t, app, "auth.google_register_enabled", "1")
	setCfg(t, app, "auth.google_client_id", "client-1")
	app.GoogleVerify = stubGoogle{ident: googleid.Identity{
		Subject: "gid-setpw", Email: "setpw@example.com", EmailVerified: true, Name: "Set PW",
	}}
	created := doJSON(t, app, http.MethodPost, "/api/v1/auth/google", "", map[string]string{
		"idToken": "tok", "client": "web",
	})
	if created.Code != http.StatusOK {
		t.Fatalf("google: %d %s", created.Code, created.Body.String())
	}
	var first struct {
		Token string  `json:"token"`
		User  userDTO `json:"user"`
	}
	if err := json.Unmarshal(decodeEnv(t, created).Data, &first); err != nil || first.Token == "" || !first.User.MustSetPassword {
		t.Fatalf("dto: %+v %s", first, created.Body.String())
	}
	unbind := doJSON(t, app, http.MethodPost, "/api/v1/auth/google/unbind", first.Token, map[string]string{
		"password": "anything12",
	})
	if unbind.Code != http.StatusBadRequest || decodeEnv(t, unbind).ErrorCode != CodeGoogleNeedPassword {
		t.Fatalf("unbind before password: %d %s", unbind.Code, unbind.Body.String())
	}
	setPW := doJSON(t, app, http.MethodPut, "/api/v1/auth/password", first.Token, map[string]string{
		"newPassword": "GooglePass12a",
	})
	if setPW.Code != http.StatusOK {
		t.Fatalf("set password: %d %s", setPW.Code, setPW.Body.String())
	}
	id, answer, _ := issueCaptcha(t, app)
	login := doJSON(t, app, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"username": first.User.Username, "password": "GooglePass12a", "client": "web",
		"captchaId": id, "captchaCode": answer,
	})
	if login.Code != http.StatusOK {
		t.Fatalf("password login: %d %s", login.Code, login.Body.String())
	}
	var second struct {
		Token string  `json:"token"`
		User  userDTO `json:"user"`
	}
	if err := json.Unmarshal(decodeEnv(t, login).Data, &second); err != nil || second.User.MustSetPassword {
		t.Fatalf("after set: %+v %s", second, login.Body.String())
	}
	denied := doJSON(t, app, http.MethodPut, "/api/v1/auth/password", second.Token, map[string]string{
		"newPassword": "GooglePass13a",
	})
	if denied.Code != http.StatusBadRequest || decodeEnv(t, denied).ErrorCode != CodePasswordRequired {
		t.Fatalf("second set without old: %d %s", denied.Code, denied.Body.String())
	}
}
