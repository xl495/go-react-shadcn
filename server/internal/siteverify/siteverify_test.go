package siteverify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
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
