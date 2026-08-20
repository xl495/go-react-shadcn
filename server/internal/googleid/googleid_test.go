package googleid

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPVerifierAcceptsValidToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("id_token") != "tok" {
			t.Fatalf("id_token=%q", r.URL.Query().Get("id_token"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"aud":            "client-1",
			"sub":            "sub-9",
			"email":          "Ada@Example.com",
			"email_verified": "true",
			"name":           "Ada",
			"picture":        "https://example.com/a.png",
			"exp":            time.Now().Add(time.Hour).Unix(),
		})
	}))
	t.Cleanup(srv.Close)

	orig := tokenInfoURL
	tokenInfoURL = srv.URL
	t.Cleanup(func() { tokenInfoURL = orig })

	ident, err := HTTPVerifier{Client: srv.Client()}.Verify(context.Background(), "tok", "client-1")
	if err != nil {
		t.Fatal(err)
	}
	if ident.Subject != "sub-9" || ident.Email != "ada@example.com" || !ident.EmailVerified || ident.Name != "Ada" {
		t.Fatalf("%+v", ident)
	}
}

func TestHTTPVerifierRejectsAudienceMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"aud": "other",
			"sub": "sub-9",
			"exp": time.Now().Add(time.Hour).Unix(),
		})
	}))
	t.Cleanup(srv.Close)
	orig := tokenInfoURL
	tokenInfoURL = srv.URL
	t.Cleanup(func() { tokenInfoURL = orig })

	if _, err := (HTTPVerifier{Client: srv.Client()}).Verify(context.Background(), "tok", "client-1"); err == nil {
		t.Fatal("expected audience error")
	}
}
