package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"go-react-shadcn/internal/googleid"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/seed"
	"go-react-shadcn/internal/siteverify"
)

type stubGoogle struct {
	ident googleid.Identity
	err   error
}

func (s stubGoogle) Verify(context.Context, string, string) (googleid.Identity, error) {
	return s.ident, s.err
}

type stubSite struct {
	result siteverify.Result
	err    error
	calls  int
}

func (s *stubSite) Check(context.Context, string, string, string, string) (siteverify.Result, error) {
	s.calls++
	return s.result, s.err
}

func setCfg(t *testing.T, app *App, key, value string) {
	t.Helper()
	if err := app.DB.Model(&models.SysConfig{}).Where(`"key" = ?`, key).Update("value", value).Error; err != nil {
		t.Fatalf("set %s: %v", key, err)
	}
	app.syscfg.invalidate()
}

func TestAuthSettingsDefaults(t *testing.T) {
	app := testApp(t)
	w := doJSON(t, app, http.MethodGet, "/api/v1/auth/settings", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d %s", w.Code, w.Body.String())
	}
	var settings publicAuthSettings
	if err := json.Unmarshal(decodeEnv(t, w).Data, &settings); err != nil {
		t.Fatal(err)
	}
	if settings.CaptchaProvider != "image" {
		t.Fatalf("provider=%q", settings.CaptchaProvider)
	}
	if settings.GoogleEnabled || settings.GoogleRegisterEnabled {
		t.Fatalf("google should be off by default: %+v", settings)
	}
}

func TestLoginNoneCaptchaSkipsChallenge(t *testing.T) {
	app := testApp(t)
	setCfg(t, app, "auth.captcha_provider", "none")
	w := doJSON(t, app, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"username": seed.AdminUsername,
		"password": seed.AdminPassword,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d %s", w.Code, w.Body.String())
	}
	if tokenFrom(t, w) == "" {
		t.Fatal("expected jwt")
	}
}

func TestRecaptchaV3LowScoreFallsBackToV2(t *testing.T) {
	app := testApp(t)
	setCfg(t, app, "auth.captcha_provider", "recaptcha")
	setCfg(t, app, "auth.recaptcha_secret_v3", "s3")
	setCfg(t, app, "auth.recaptcha_secret_v2", "s2")
	setCfg(t, app, "auth.recaptcha_site_key_v2", "site-v2")
	setCfg(t, app, "auth.recaptcha_min_score", "0.5")
	app.SiteVerify = &stubSite{result: siteverify.Result{Success: true, Score: 0.1, Action: "login"}}

	w := doJSON(t, app, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"username":       seed.AdminUsername,
		"password":       seed.AdminPassword,
		"captchaToken":   "v3-token",
		"captchaVersion": "v3",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d %s", w.Code, w.Body.String())
	}
	if decodeEnv(t, w).ErrorCode != CodeCaptchaFallback {
		t.Fatalf("code=%d body=%s", decodeEnv(t, w).ErrorCode, w.Body.String())
	}
}

func TestRecaptchaV2LoginOK(t *testing.T) {
	app := testApp(t)
	setCfg(t, app, "auth.captcha_provider", "recaptcha")
	setCfg(t, app, "auth.recaptcha_secret_v3", "s3")
	setCfg(t, app, "auth.recaptcha_secret_v2", "s2")
	app.SiteVerify = &stubSite{result: siteverify.Result{Success: true}}

	w := doJSON(t, app, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"username":       seed.AdminUsername,
		"password":       seed.AdminPassword,
		"captchaToken":   "v2-token",
		"captchaVersion": "v2",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d %s", w.Code, w.Body.String())
	}
}

func TestTurnstileLoginOK(t *testing.T) {
	app := testApp(t)
	setCfg(t, app, "auth.captcha_provider", "turnstile")
	setCfg(t, app, "auth.turnstile_secret", "cf-secret")
	app.SiteVerify = &stubSite{result: siteverify.Result{Success: true}}

	w := doJSON(t, app, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"username":     seed.AdminUsername,
		"password":     seed.AdminPassword,
		"captchaToken": "cf-token",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d %s", w.Code, w.Body.String())
	}
}

func probeTurnstile(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	_, err := (&siteverify.Client{HTTP: &http.Client{Timeout: 8 * time.Second}}).Check(ctx, siteverify.Turnstile, siteverify.TurnstileDummyPassSecret, siteverify.TurnstileDummyToken, "127.0.0.1")
	if err != nil {
		t.Skip("cloudflare turnstile unreachable: ", err)
	}
}

func TestTurnstileDummyPassSecretLogin(t *testing.T) {
	probeTurnstile(t)
	app := testApp(t)
	setCfg(t, app, "auth.captcha_provider", "turnstile")
	setCfg(t, app, "auth.turnstile_site_key", siteverify.TurnstileDummyPassSiteKey)
	setCfg(t, app, "auth.turnstile_secret", siteverify.TurnstileDummyPassSecret)

	w := doJSON(t, app, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"username":     seed.AdminUsername,
		"password":     seed.AdminPassword,
		"captchaToken": siteverify.TurnstileDummyToken,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("dummy pass login: %d %s", w.Code, w.Body.String())
	}
	if tokenFrom(t, w) == "" {
		t.Fatal("expected jwt")
	}
}

func TestTurnstileDummyFailSecretLogin(t *testing.T) {
	probeTurnstile(t)
	app := testApp(t)
	setCfg(t, app, "auth.captcha_provider", "turnstile")
	setCfg(t, app, "auth.turnstile_secret", siteverify.TurnstileDummyFailSecret)

	w := doJSON(t, app, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"username":     seed.AdminUsername,
		"password":     seed.AdminPassword,
		"captchaToken": siteverify.TurnstileDummyToken,
	})
	if w.Code != http.StatusBadRequest || decodeEnv(t, w).ErrorCode != CodeInvalidCaptcha {
		t.Fatalf("dummy fail login: %d %s", w.Code, w.Body.String())
	}
}

func TestTurnstileDummySpentSecretLogin(t *testing.T) {
	probeTurnstile(t)
	app := testApp(t)
	setCfg(t, app, "auth.captcha_provider", "turnstile")
	setCfg(t, app, "auth.turnstile_secret", siteverify.TurnstileDummySpentSecret)

	w := doJSON(t, app, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"username":     seed.AdminUsername,
		"password":     seed.AdminPassword,
		"captchaToken": siteverify.TurnstileDummyToken,
	})
	if w.Code != http.StatusBadRequest || decodeEnv(t, w).ErrorCode != CodeInvalidCaptcha {
		t.Fatalf("dummy spent login: %d %s", w.Code, w.Body.String())
	}
}

func TestAuthSettingsExposesTurnstileDummySiteKey(t *testing.T) {
	app := testApp(t)
	setCfg(t, app, "auth.captcha_provider", "turnstile")
	setCfg(t, app, "auth.turnstile_site_key", siteverify.TurnstileDummyPassSiteKey)
	setCfg(t, app, "auth.turnstile_secret", siteverify.TurnstileDummyPassSecret)
	w := doJSON(t, app, http.MethodGet, "/api/v1/auth/settings", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("settings: %d %s", w.Code, w.Body.String())
	}
	var settings publicAuthSettings
	if err := json.Unmarshal(decodeEnv(t, w).Data, &settings); err != nil {
		t.Fatal(err)
	}
	if settings.CaptchaProvider != "turnstile" || settings.TurnstileSiteKey != siteverify.TurnstileDummyPassSiteKey {
		t.Fatalf("%+v", settings)
	}
	if strings.Contains(w.Body.String(), siteverify.TurnstileDummyPassSecret) {
		t.Fatal("turnstile secret leaked in public settings")
	}
}

func TestGoogleRegisterAndLogin(t *testing.T) {
	app := testApp(t)
	setCfg(t, app, "auth.google_enabled", "1")
	setCfg(t, app, "auth.google_register_enabled", "1")
	setCfg(t, app, "auth.google_client_id", "client-1")
	app.GoogleVerify = stubGoogle{ident: googleid.Identity{
		Subject: "gid-1", Email: "new.user@example.com", EmailVerified: true, Name: "New User",
	}}

	created := doJSON(t, app, http.MethodPost, "/api/v1/auth/google", "", map[string]string{
		"idToken": "tok", "client": "web",
	})
	if created.Code != http.StatusOK {
		t.Fatalf("register status=%d %s", created.Code, created.Body.String())
	}
	var first struct {
		Token string `json:"token"`
		User  struct {
			Username string `json:"username"`
			Kind     string `json:"kind"`
			Email    string `json:"email"`
		} `json:"user"`
	}
	if err := json.Unmarshal(decodeEnv(t, created).Data, &first); err != nil {
		t.Fatal(err)
	}
	if first.Token == "" || first.User.Kind != models.UserKindWeb || first.User.Email != "new.user@example.com" {
		t.Fatalf("%+v", first)
	}

	again := doJSON(t, app, http.MethodPost, "/api/v1/auth/google", "", map[string]string{
		"idToken": "tok", "client": "web",
	})
	if again.Code != http.StatusOK {
		t.Fatalf("login status=%d %s", again.Code, again.Body.String())
	}
	var second struct {
		User struct {
			Username string `json:"username"`
		} `json:"user"`
	}
	if err := json.Unmarshal(decodeEnv(t, again).Data, &second); err != nil {
		t.Fatal(err)
	}
	if second.User.Username != first.User.Username {
		t.Fatalf("username changed %q -> %q", first.User.Username, second.User.Username)
	}

	blocked := doJSON(t, app, http.MethodPost, "/api/v1/auth/google", "", map[string]string{
		"idToken": "tok", "client": "admin",
	})
	if blocked.Code != http.StatusForbidden || decodeEnv(t, blocked).ErrorCode != CodeGoogleRegisterDisabled {
		t.Fatalf("admin client want register disabled, got %d %s", blocked.Code, blocked.Body.String())
	}
}

func TestGoogleRegisterDisabled(t *testing.T) {
	app := testApp(t)
	setCfg(t, app, "auth.google_enabled", "1")
	setCfg(t, app, "auth.google_register_enabled", "0")
	setCfg(t, app, "auth.google_client_id", "client-1")
	app.GoogleVerify = stubGoogle{ident: googleid.Identity{
		Subject: "gid-2", Email: "nobody@example.com", EmailVerified: true, Name: "N",
	}}
	w := doJSON(t, app, http.MethodPost, "/api/v1/auth/google", "", map[string]string{
		"idToken": "tok", "client": "web",
	})
	if w.Code != http.StatusForbidden || decodeEnv(t, w).ErrorCode != CodeGoogleRegisterDisabled {
		t.Fatalf("got %d %s", w.Code, w.Body.String())
	}
}

func TestGoogleLinksExistingEmail(t *testing.T) {
	app := testApp(t)
	setCfg(t, app, "auth.google_enabled", "1")
	setCfg(t, app, "auth.google_client_id", "client-1")
	app.GoogleVerify = stubGoogle{ident: googleid.Identity{
		Subject: "gid-admin", Email: "admin@latch.local", EmailVerified: true, Name: "系统管理员",
	}}
	w := doJSON(t, app, http.MethodPost, "/api/v1/auth/google", "", map[string]string{
		"idToken": "tok", "client": "admin",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d %s", w.Code, w.Body.String())
	}
	var user models.User
	if err := app.DB.Where("username = ?", seed.AdminUsername).First(&user).Error; err != nil {
		t.Fatal(err)
	}
	if user.GoogleID != "gid-admin" {
		t.Fatalf("google_id=%q", user.GoogleID)
	}
}

func TestGoogleDisabledWithoutClientID(t *testing.T) {
	app := testApp(t)
	setCfg(t, app, "auth.google_enabled", "1")
	w := doJSON(t, app, http.MethodPost, "/api/v1/auth/google", "", map[string]string{
		"idToken": "tok", "client": "web",
	})
	if w.Code != http.StatusServiceUnavailable || decodeEnv(t, w).ErrorCode != CodeGoogleDisabled {
		t.Fatalf("got %d %s", w.Code, w.Body.String())
	}
}

func TestSeededAuthConfigsListed(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	listed := doJSON(t, app, http.MethodGet, "/api/v1/configs?group=auth&pageSize=50", admin, nil)
	if listed.Code != http.StatusOK {
		t.Fatalf("list: %d %s", listed.Code, listed.Body.String())
	}
	items := decodePage[models.SysConfig](t, listed).Items
	got := map[string]models.SysConfig{}
	for _, row := range items {
		got[row.Key] = row
	}
	for _, key := range []string{
		"auth.google_enabled",
		"auth.google_register_enabled",
		"auth.google_client_id",
		"auth.google_client_secret",
		"auth.captcha_provider",
		"auth.recaptcha_site_key_v3",
		"auth.recaptcha_secret_v3",
		"auth.recaptcha_site_key_v2",
		"auth.recaptcha_secret_v2",
		"auth.recaptcha_min_score",
		"auth.turnstile_site_key",
		"auth.turnstile_secret",
	} {
		if _, ok := got[key]; !ok {
			t.Fatalf("missing seeded config %s (got %d auth rows)", key, len(items))
		}
	}
	if got["auth.captcha_provider"].Value != "image" {
		t.Fatalf("captcha_provider=%q", got["auth.captcha_provider"].Value)
	}
	if got["auth.google_enabled"].Value != "0" {
		t.Fatalf("google_enabled=%q", got["auth.google_enabled"].Value)
	}
}

func TestCreateAuthConfigWhenMissing(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	var row models.SysConfig
	if err := app.DB.Where(`"key" = ?`, "auth.google_enabled").First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if err := app.DB.Delete(&row).Error; err != nil {
		t.Fatal(err)
	}
	created := doJSON(t, app, http.MethodPost, "/api/v1/configs", admin, map[string]string{
		"key": "auth.google_enabled", "value": "1", "name": "Google 登录", "group": "auth",
	})
	if created.Code != http.StatusOK {
		t.Fatalf("create: %d %s", created.Code, created.Body.String())
	}
	listed := doJSON(t, app, http.MethodGet, "/api/v1/configs?group=auth&pageSize=50", admin, nil)
	if listed.Code != http.StatusOK {
		t.Fatalf("list: %d %s", listed.Code, listed.Body.String())
	}
	found := false
	for _, item := range decodePage[models.SysConfig](t, listed).Items {
		if item.Key == "auth.google_enabled" && item.Value == "1" {
			found = true
		}
	}
	if !found {
		t.Fatal("created auth.google_enabled not listed under group=auth")
	}
}

func TestAuthSecretsRedacted(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	var row models.SysConfig
	if err := app.DB.Where(`"key" = ?`, "auth.turnstile_secret").First(&row).Error; err != nil {
		t.Fatal(err)
	}
	upd := doJSON(t, app, http.MethodPut, "/api/v1/configs/"+itoa(row.ID), admin, map[string]string{
		"value": "cf-super-secret", "name": row.Name, "group": row.Group, "remark": row.Remark,
	})
	if upd.Code != http.StatusOK {
		t.Fatalf("update: %d %s", upd.Code, upd.Body.String())
	}
	listed := doJSON(t, app, http.MethodGet, "/api/v1/configs?group=auth&pageSize=50", admin, nil)
	if listed.Code != http.StatusOK {
		t.Fatalf("list: %d %s", listed.Code, listed.Body.String())
	}
	if strings.Contains(listed.Body.String(), "cf-super-secret") {
		t.Fatal("turnstile secret leaked")
	}
}
