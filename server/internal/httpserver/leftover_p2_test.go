package httpserver

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/seed"
)

func TestRegisterWebUserAndRejectWhenClosed(t *testing.T) {
	app := testApp(t)
	id, answer, _ := issueCaptcha(t, app)
	created := doJSON(t, app, http.MethodPost, "/api/v1/auth/register", "", map[string]string{
		"username": "newbie", "email": "newbie@example.com", "password": "newbie12",
		"client": "web", "captchaId": id, "captchaCode": answer,
	})
	if created.Code != http.StatusOK {
		t.Fatalf("register: %d %s", created.Code, created.Body.String())
	}
	var pending struct {
		Pending     bool   `json:"pending"`
		VerifyToken string `json:"verifyToken"`
	}
	if err := json.Unmarshal(decodeEnv(t, created).Data, &pending); err != nil || !pending.Pending || pending.VerifyToken == "" {
		t.Fatalf("register pending: %s", created.Body.String())
	}
	var row models.User
	if err := app.accounts(models.UserKindWeb).Where("username = ?", "newbie").First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.MarketingOptIn || row.EmailVerified {
		t.Fatal("new register should opt out and stay unverified")
	}
	idLogin, answerLogin, _ := issueCaptcha(t, app)
	blocked := doJSON(t, app, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"username": "newbie", "password": "newbie12", "client": "web",
		"captchaId": idLogin, "captchaCode": answerLogin,
	})
	if blocked.Code != http.StatusForbidden || decodeEnv(t, blocked).ErrorCode != CodeEmailUnverified {
		t.Fatalf("unverified login: %d %s", blocked.Code, blocked.Body.String())
	}
	verified := doJSON(t, app, http.MethodPost, "/api/v1/auth/verify-email", "", map[string]string{
		"token": pending.VerifyToken,
	})
	if verified.Code != http.StatusOK {
		t.Fatalf("verify: %d %s", verified.Code, verified.Body.String())
	}

	adminAttempt := doJSON(t, app, http.MethodPost, "/api/v1/auth/register", "", map[string]string{
		"username": "staff", "email": "staff@example.com", "password": "staff1234",
		"client": "admin",
	})
	if adminAttempt.Code != http.StatusForbidden {
		t.Fatalf("admin client register: %d %s", adminAttempt.Code, adminAttempt.Body.String())
	}

	setCfg(t, app, "auth.register_enabled", "0")
	id2, answer2, _ := issueCaptcha(t, app)
	closed := doJSON(t, app, http.MethodPost, "/api/v1/auth/register", "", map[string]string{
		"username": "later", "email": "later@example.com", "password": "later1234",
		"client": "web", "captchaId": id2, "captchaCode": answer2,
	})
	if closed.Code != http.StatusForbidden || decodeEnv(t, closed).ErrorCode != CodeRegisterDisabled {
		t.Fatalf("closed register: %d %s", closed.Code, closed.Body.String())
	}
}

func TestTraceIDRejectsNonUUID(t *testing.T) {
	app := testApp(t)
	token := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	forged := traceMe(t, app, token, "not-a-uuid")
	if forged == "not-a-uuid" {
		t.Fatal("accepted forged trace id")
	}
	okID := "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"
	if got := traceMe(t, app, token, okID); got != okID {
		t.Fatalf("uuid trace=%q", got)
	}
}

func traceMe(t *testing.T, app *App, token, trace string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Trace-Id", trace)
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("me: %d %s", w.Code, w.Body.String())
	}
	return w.Header().Get("X-Trace-Id")
}

func TestGzipJSONWhenAsked(t *testing.T) {
	app := testApp(t)
	req := httptest.NewRequest(http.MethodGet, "/live", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("live: %d %s", w.Code, w.Body.String())
	}
	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("encoding=%q", w.Header().Get("Content-Encoding"))
	}
	zr, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = zr.Close() }()
	body, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(body) {
		t.Fatalf("gunzipped body: %s", body)
	}
}

func TestReadyReportsDB(t *testing.T) {
	app := testApp(t)
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"status":"ready"`) {
		t.Fatalf("ready: %d %s", w.Code, w.Body.String())
	}
}
