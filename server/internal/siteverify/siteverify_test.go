package siteverify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientCheckParsesScore(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if string(body) == "" || r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Fatalf("bad request %s %s", r.Header.Get("Content-Type"), body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"score":   0.9,
			"action":  "login",
		})
	}))
	t.Cleanup(srv.Close)
	out, err := (&Client{HTTP: srv.Client()}).Check(context.Background(), srv.URL, "secret", "token", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if !out.Success || out.Score < 0.89 || out.Action != "login" {
		t.Fatalf("%+v", out)
	}
}

func TestClientCheckMissingSecret(t *testing.T) {
	if _, err := (&Client{}).Check(context.Background(), Recaptcha, "", "token", ""); err == nil {
		t.Fatal("expected error")
	}
}

func liveTurnstile(t *testing.T, secret, token string) Result {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	out, err := (&Client{HTTP: &http.Client{Timeout: 8 * time.Second}}).Check(ctx, Turnstile, secret, token, "127.0.0.1")
	if err != nil {
		t.Skip("cloudflare turnstile unreachable: ", err)
	}
	return out
}

func TestTurnstileDummySecrets(t *testing.T) {
	pass := liveTurnstile(t, TurnstileDummyPassSecret, TurnstileDummyToken)
	if !pass.Success {
		t.Fatalf("dummy pass secret: %+v", pass)
	}

	fail := liveTurnstile(t, TurnstileDummyFailSecret, TurnstileDummyToken)
	if fail.Success {
		t.Fatalf("dummy fail secret should reject: %+v", fail)
	}

	spent := liveTurnstile(t, TurnstileDummySpentSecret, TurnstileDummyToken)
	if spent.Success {
		t.Fatalf("dummy spent secret should reject: %+v", spent)
	}
	foundSpent := false
	for _, code := range spent.ErrorCodes {
		if code == "timeout-or-duplicate" {
			foundSpent = true
		}
	}
	if !foundSpent {
		t.Fatalf("expected timeout-or-duplicate, got %+v", spent)
	}

	// Dummy pass secret is "always verify success" — it accepts placeholder tokens too.
	anyToken := liveTurnstile(t, TurnstileDummyPassSecret, "02.not-a-dummy-token")
	if !anyToken.Success {
		t.Fatalf("dummy pass secret should accept any token: %+v", anyToken)
	}
}
