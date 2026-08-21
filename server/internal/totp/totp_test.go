package totp

import (
	"testing"
	"time"
)

func TestRoundTrip(t *testing.T) {
	secret, err := RandomSecret()
	if err != nil || secret == "" {
		t.Fatalf("secret: %v %q", err, secret)
	}
	now := time.Now()
	code, err := Code(secret, now)
	if err != nil {
		t.Fatal(err)
	}
	if !Valid(secret, code, now) {
		t.Fatalf("expected %s to be valid", code)
	}
	if Valid(secret, "000000", now) {
		t.Fatal("zero code should fail")
	}
	if !stringsHasPrefix(URI("admin", secret), "otpauth://totp/") {
		t.Fatal(URI("admin", secret))
	}
}

func stringsHasPrefix(s, p string) bool {
	return len(s) >= len(p) && s[:len(p)] == p
}
