package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"go-react-shadcn/internal/mailer"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/seed"
)

func TestMailConfigsRedactPassword(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	listed := doJSON(t, app, http.MethodGet, "/api/v1/configs?group=mail&pageSize=50", admin, nil)
	if listed.Code != http.StatusOK {
		t.Fatalf("list mail configs: %d %s", listed.Code, listed.Body.String())
	}
	if !strings.Contains(listed.Body.String(), `"mail.host"`) {
		t.Fatalf("expected seeded mail configs: %s", listed.Body.String())
	}

	var row models.SysConfig
	if err := app.DB.Where(`"key" = ?`, "mail.password").First(&row).Error; err != nil {
		t.Fatal(err)
	}
	upd := doJSON(t, app, http.MethodPut, "/api/v1/configs/"+itoa(row.ID), admin, map[string]string{
		"value": "super-secret", "name": row.Name, "group": "mail", "remark": row.Remark,
	})
	if upd.Code != http.StatusOK {
		t.Fatalf("update password: %d %s", upd.Code, upd.Body.String())
	}
	if strings.Contains(upd.Body.String(), "super-secret") {
		t.Fatal("password leaked in update response")
	}
	listed = doJSON(t, app, http.MethodGet, "/api/v1/configs?group=mail&pageSize=50", admin, nil)
	if strings.Contains(listed.Body.String(), "super-secret") {
		t.Fatal("password leaked in list")
	}
	if !strings.Contains(listed.Body.String(), mailer.SecretMask) {
		t.Fatalf("expected mask in list: %s", listed.Body.String())
	}
	keep := doJSON(t, app, http.MethodPut, "/api/v1/configs/"+itoa(row.ID), admin, map[string]string{
		"value": mailer.SecretMask, "name": row.Name, "group": "mail",
	})
	if keep.Code != http.StatusOK {
		t.Fatalf("keep secret: %d", keep.Code)
	}
	var stored models.SysConfig
	if err := app.DB.First(&stored, row.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Value != "super-secret" {
		t.Fatalf("stored password=%q", stored.Value)
	}
}

func useMemoryMail(t *testing.T, app *App) *mailer.Memory {
	t.Helper()
	mem := &mailer.Memory{}
	app.Mail = mem
	app.MailQ.SetSender(mem)
	if err := app.DB.Model(&models.SysConfig{}).Where(`"key" = ?`, "mail.enabled").Update("value", "1").Error; err != nil {
		t.Fatal(err)
	}
	return mem
}

func TestForgotAndResetPassword(t *testing.T) {
	app := testApp(t)
	mem := useMemoryMail(t, app)

	id, answer, _ := issueCaptcha(t, app)
	unknown := doJSON(t, app, http.MethodPost, "/api/v1/auth/forgot-password", "", map[string]string{
		"email": "nobody@latch.local", "captchaId": id, "captchaCode": answer,
	})
	if unknown.Code != http.StatusOK {
		t.Fatalf("unknown email: %d %s", unknown.Code, unknown.Body.String())
	}
	if _, ok := mem.Last(); ok {
		t.Fatal("must not send mail for unknown address")
	}

	id, answer, _ = issueCaptcha(t, app)
	forgot := doJSON(t, app, http.MethodPost, "/api/v1/auth/forgot-password", "", map[string]string{
		"email": "Admin@latch.local", "captchaId": id, "captchaCode": answer,
	})
	if forgot.Code != http.StatusOK {
		t.Fatalf("forgot: %d %s", forgot.Code, forgot.Body.String())
	}
	msg, ok := mem.Last()
	if !ok || !strings.Contains(msg.To, "admin@latch.local") {
		t.Fatalf("expected reset mail, got %+v", msg)
	}
	token := tokenFromResetMail(t, msg.Body)

	bad := doJSON(t, app, http.MethodPost, "/api/v1/auth/reset-password", "", map[string]string{
		"token": token, "newPassword": "short",
	})
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("short password: %d", bad.Code)
	}

	reset := doJSON(t, app, http.MethodPost, "/api/v1/auth/reset-password", "", map[string]string{
		"token": token, "newPassword": "admin-reset-1",
	})
	if reset.Code != http.StatusOK {
		t.Fatalf("reset: %d %s", reset.Code, reset.Body.String())
	}
	reuse := doJSON(t, app, http.MethodPost, "/api/v1/auth/reset-password", "", map[string]string{
		"token": token, "newPassword": "admin-reset-2",
	})
	if reuse.Code != http.StatusBadRequest {
		t.Fatalf("reuse token: %d %s", reuse.Code, reuse.Body.String())
	}

	id, ans := mustCaptcha(t, app)
	if w := login(t, app, seed.AdminUsername, seed.AdminPassword, id, ans); w.Code == http.StatusOK {
		t.Fatal("old password should fail")
	}
	if loginOK(t, app, seed.AdminUsername, "admin-reset-1") == "" {
		t.Fatal("new password should work")
	}
}

func TestExpiredResetToken(t *testing.T) {
	app := testApp(t)
	var user models.User
	if err := app.DB.Where("username = ?", seed.ViewerUsername).First(&user).Error; err != nil {
		t.Fatal(err)
	}
	raw, hash, err := mailer.NewResetToken()
	if err != nil {
		t.Fatal(err)
	}
	expired := time.Now().Add(-time.Minute)
	if err := app.DB.Create(&models.PasswordResetToken{
		UserID: user.ID, TokenHash: hash, ExpiresAt: expired,
	}).Error; err != nil {
		t.Fatal(err)
	}
	w := doJSON(t, app, http.MethodPost, "/api/v1/auth/reset-password", "", map[string]string{
		"token": raw, "newPassword": "viewer-new-1",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expired token: %d %s", w.Code, w.Body.String())
	}
}

func TestMailTestEndpoint(t *testing.T) {
	app := testApp(t)
	mem := useMemoryMail(t, app)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	operator := loginOK(t, app, seed.OperatorUsername, seed.OperatorPassword)

	if w := doJSON(t, app, http.MethodPost, "/api/v1/mail/test", operator, map[string]string{
		"to": "ops@example.com",
	}); w.Code != http.StatusForbidden {
		t.Fatalf("operator test mail: %d %s", w.Code, w.Body.String())
	}

	mem.Disabled = true
	if w := doJSON(t, app, http.MethodPost, "/api/v1/mail/test", admin, map[string]string{
		"to": "ops@example.com",
	}); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("disabled mail: %d %s", w.Code, w.Body.String())
	}
	mem.Disabled = false
	okSend := doJSON(t, app, http.MethodPost, "/api/v1/mail/test", admin, map[string]string{
		"to": "ops@example.com",
	})
	if okSend.Code != http.StatusOK {
		t.Fatalf("admin test mail: %d %s", okSend.Code, okSend.Body.String())
	}
	msg, ok := mem.Last()
	if !ok || msg.To != "ops@example.com" {
		t.Fatalf("captured %+v", msg)
	}
}

func mustCaptcha(t *testing.T, app *App) (id, answer string) {
	t.Helper()
	id, answer, _ = issueCaptcha(t, app)
	return id, answer
}

func tokenFromResetMail(t *testing.T, body string) string {
	t.Helper()
	const mark = "token="
	i := strings.Index(body, mark)
	if i < 0 {
		t.Fatalf("token missing in mail: %s", body)
	}
	rest := body[i+len(mark):]
	rest = strings.TrimSpace(rest)
	if n := strings.IndexAny(rest, "\r\n "); n >= 0 {
		rest = rest[:n]
	}
	if rest == "" {
		t.Fatal("empty token")
	}
	return rest
}

func TestForgotRequiresCaptcha(t *testing.T) {
	app := testApp(t)
	w := doJSON(t, app, http.MethodPost, "/api/v1/auth/forgot-password", "", map[string]string{
		"email": "admin@latch.local",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("captcha required: %d %s", w.Code, w.Body.String())
	}
	var env envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.Code != CodeCaptchaRequired {
		t.Fatalf("code=%d", env.Code)
	}
}

func TestMailJobsRetryCancelAndCampaigns(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	operator := loginOK(t, app, seed.OperatorUsername, seed.OperatorPassword)

	dead := models.MailJob{
		Class: models.MailClassTransactional, Priority: 1, ToEmail: "dead@example.com",
		Timezone: "Asia/Shanghai", Subject: "dead", Body: "x", Status: models.MailStatusDead,
		SendAfter: time.Now(), LastError: "boom",
	}
	if err := app.DB.Create(&dead).Error; err != nil {
		t.Fatal(err)
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/mail/jobs/"+itoa(dead.ID)+"/retry", operator, nil); w.Code != http.StatusForbidden {
		t.Fatalf("operator retry: %d %s", w.Code, w.Body.String())
	}
	retry := doJSON(t, app, http.MethodPost, "/api/v1/mail/jobs/"+itoa(dead.ID)+"/retry", admin, nil)
	if retry.Code != http.StatusOK {
		t.Fatalf("retry: %d %s", retry.Code, retry.Body.String())
	}
	listed := doJSON(t, app, http.MethodGet, "/api/v1/mail/jobs?status=queued", admin, nil)
	if listed.Code != http.StatusOK {
		t.Fatalf("list jobs: %d %s", listed.Code, listed.Body.String())
	}
	jobs := decodePage[models.MailJob](t, listed).Items
	if len(jobs) == 0 {
		t.Fatal("expected queued job after retry")
	}
	cancel := doJSON(t, app, http.MethodPost, "/api/v1/mail/jobs/"+itoa(dead.ID)+"/cancel", admin, nil)
	if cancel.Code != http.StatusOK {
		t.Fatalf("cancel: %d %s", cancel.Code, cancel.Body.String())
	}

	useMemoryMail(t, app)

	created := doJSON(t, app, http.MethodPost, "/api/v1/mail/campaigns", admin, map[string]string{
		"name": "Autumn", "subject": "Hello {{nickname}}", "body": "Hi {{username}}", "audience": "opted_in",
	})
	if created.Code != http.StatusOK {
		t.Fatalf("create campaign: %d %s", created.Code, created.Body.String())
	}
	var camp models.MailCampaign
	if err := json.Unmarshal(decodeEnv(t, created).Data, &camp); err != nil {
		t.Fatal(err)
	}
	got := doJSON(t, app, http.MethodGet, "/api/v1/mail/campaigns/"+itoa(camp.ID), admin, nil)
	if got.Code != http.StatusOK {
		t.Fatalf("get campaign: %d %s", got.Code, got.Body.String())
	}
	if w := doJSON(t, app, http.MethodPost, "/api/v1/mail/campaigns", operator, map[string]string{
		"name": "nope", "subject": "x",
	}); w.Code != http.StatusForbidden {
		t.Fatalf("operator create campaign: %d", w.Code)
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/mail/campaigns", operator, nil); w.Code != http.StatusOK {
		t.Fatalf("operator list campaigns: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, app, http.MethodGet, "/api/v1/mail/campaigns/"+itoa(camp.ID), operator, nil); w.Code != http.StatusOK {
		t.Fatalf("operator get campaign: %d %s", w.Code, w.Body.String())
	}

	later := doJSON(t, app, http.MethodPost, "/api/v1/mail/campaigns", admin, map[string]string{
		"name": "Later", "subject": "Hold", "body": "wait",
	})
	var held models.MailCampaign
	if err := json.Unmarshal(decodeEnv(t, later).Data, &held); err != nil {
		t.Fatal(err)
	}
	when := time.Now().Add(2 * time.Hour)
	heldSched := doJSON(t, app, http.MethodPost, "/api/v1/mail/campaigns/"+itoa(held.ID)+"/schedule", admin, map[string]any{
		"scheduledAt": when.UTC().Format(time.RFC3339),
	})
	if heldSched.Code != http.StatusOK {
		t.Fatalf("future schedule: %d %s", heldSched.Code, heldSched.Body.String())
	}
	paused := doJSON(t, app, http.MethodPut, "/api/v1/mail/campaigns/"+itoa(held.ID), admin, map[string]string{
		"status": "paused",
	})
	if paused.Code != http.StatusOK {
		t.Fatalf("pause: %d %s", paused.Code, paused.Body.String())
	}

	sched := doJSON(t, app, http.MethodPost, "/api/v1/mail/campaigns/"+itoa(camp.ID)+"/schedule", admin, map[string]any{})
	if sched.Code != http.StatusOK {
		t.Fatalf("schedule: %d %s", sched.Code, sched.Body.String())
	}
	if err := json.Unmarshal(decodeEnv(t, sched).Data, &camp); err != nil {
		t.Fatal(err)
	}
	if camp.JobCount < 3 {
		t.Fatalf("expected fan-out jobs, jobCount=%d status=%s", camp.JobCount, camp.Status)
	}
}

func TestUnsubscribeAndUserTimezone(t *testing.T) {
	app := testApp(t)
	var user models.User
	if err := app.DB.Where("username = ?", seed.ViewerUsername).First(&user).Error; err != nil {
		t.Fatal(err)
	}
	tok := mailer.UnsubToken(app.Cfg.JWTSecret, user.ID)
	bad := doJSON(t, app, http.MethodPost, "/api/v1/mail/unsubscribe", "", map[string]string{"token": "nope"})
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("bad token: %d %s", bad.Code, bad.Body.String())
	}
	unsub := doJSON(t, app, http.MethodPost, "/api/v1/mail/unsubscribe", "", map[string]string{"token": tok})
	if unsub.Code != http.StatusOK {
		t.Fatalf("unsub: %d %s", unsub.Code, unsub.Body.String())
	}
	if err := app.DB.First(&user, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if user.MarketingOptIn {
		t.Fatal("expected opt-out")
	}

	viewer := loginOK(t, app, seed.ViewerUsername, seed.ViewerPassword)
	upd := doJSON(t, app, http.MethodPut, "/api/v1/auth/profile", viewer, map[string]any{
		"nickname": "李访客", "email": "viewer@latch.local", "phone": "13800000003",
		"gender": "female", "department": "market", "title": "观察员",
		"timezone": "Asia/Tokyo", "marketingOptIn": false,
	})
	if upd.Code != http.StatusOK {
		t.Fatalf("timezone profile: %d %s", upd.Code, upd.Body.String())
	}
	var after userDTO
	if err := json.Unmarshal(decodeEnv(t, upd).Data, &after); err != nil {
		t.Fatal(err)
	}
	if after.Timezone != "Asia/Tokyo" || after.MarketingOptIn {
		t.Fatalf("profile tz/opt-in: %+v", after)
	}
	badTZ := doJSON(t, app, http.MethodPut, "/api/v1/auth/profile", viewer, map[string]any{
		"nickname": "李访客", "timezone": "Not/AZone",
	})
	if badTZ.Code != http.StatusBadRequest {
		t.Fatalf("invalid tz: %d %s", badTZ.Code, badTZ.Body.String())
	}
}
