package httpserver

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/security"
	"go-react-shadcn/internal/seed"
)

func TestResetAndUnsubRateLimited(t *testing.T) {
	app := testApp(t)
	app.ResetGuard = security.NewIPLimiter(2, time.Minute)
	app.ResetTokenGuard = security.NewIPLimiter(100, time.Minute)
	app.UnsubGuard = security.NewIPLimiter(2, time.Minute)

	first := doJSON(t, app, http.MethodPost, "/api/v1/auth/reset-password", "", map[string]string{
		"token": "nope", "newPassword": "reset-pass-9",
	})
	if first.Code != http.StatusBadRequest {
		t.Fatalf("first reset: %d %s", first.Code, first.Body.String())
	}
	second := doJSON(t, app, http.MethodPost, "/api/v1/auth/reset-password", "", map[string]string{
		"token": "nope", "newPassword": "reset-pass-9",
	})
	if second.Code != http.StatusBadRequest {
		t.Fatalf("second reset: %d %s", second.Code, second.Body.String())
	}
	blocked := doJSON(t, app, http.MethodPost, "/api/v1/auth/reset-password", "", map[string]string{
		"token": "nope", "newPassword": "reset-pass-9",
	})
	if blocked.Code != http.StatusTooManyRequests || decodeEnv(t, blocked).ErrorCode != CodeForgotRateLimited {
		t.Fatalf("reset limit: %d %s", blocked.Code, blocked.Body.String())
	}

	ok1 := doJSON(t, app, http.MethodPost, "/api/v1/mail/unsubscribe", "", map[string]string{"token": "bad"})
	if ok1.Code != http.StatusBadRequest {
		t.Fatalf("first unsub: %d", ok1.Code)
	}
	ok2 := doJSON(t, app, http.MethodPost, "/api/v1/mail/unsubscribe", "", map[string]string{"token": "bad"})
	if ok2.Code != http.StatusBadRequest {
		t.Fatalf("second unsub: %d", ok2.Code)
	}
	limited := doJSON(t, app, http.MethodPost, "/api/v1/mail/unsubscribe", "", map[string]string{"token": "bad"})
	if limited.Code != http.StatusTooManyRequests || decodeEnv(t, limited).ErrorCode != CodeUnsubRateLimited {
		t.Fatalf("unsub limit: %d %s", limited.Code, limited.Body.String())
	}
}

func TestMailJobListOmitsBody(t *testing.T) {
	app := testApp(t)
	admin := loginOK(t, app, seed.AdminUsername, seed.AdminPassword)
	body := strings.Repeat("<p>huge-mail-body</p>", 200)
	job := models.MailJob{
		Class: models.MailClassTransactional, Priority: 1, ToEmail: "list@example.com",
		Timezone: "Asia/Shanghai", Subject: "hi", Body: body, Status: models.MailStatusQueued,
		SendAfter: time.Now(),
	}
	if err := app.DB.Create(&job).Error; err != nil {
		t.Fatal(err)
	}
	listed := doJSON(t, app, http.MethodGet, "/api/v1/mail/jobs", admin, nil)
	if listed.Code != http.StatusOK {
		t.Fatalf("list: %d %s", listed.Code, listed.Body.String())
	}
	if strings.Contains(listed.Body.String(), "huge-mail-body") {
		t.Fatal("list leaked mail body")
	}
}
