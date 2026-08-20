package mailer

import (
	"errors"
	"testing"
)

func TestValidAddress(t *testing.T) {
	if !ValidAddress("admin@latch.local") {
		t.Fatal("expected valid")
	}
	for _, s := range []string{"", "nope", "@x", "a@", "a b@x.com"} {
		if ValidAddress(s) {
			t.Fatalf("expected invalid: %q", s)
		}
	}
}

func TestLooksSecretAndKeep(t *testing.T) {
	if !LooksSecret("mail.password") || LooksSecret("mail.host") {
		t.Fatal("secret detection")
	}
	if KeepSecret("mail.password", SecretMask, "real") != "real" {
		t.Fatal("mask should keep current")
	}
	if KeepSecret("mail.host", "smtp.example", "old") != "smtp.example" {
		t.Fatal("non-secret should update")
	}
}

func TestResetLinkAndToken(t *testing.T) {
	raw, hash, err := NewResetToken()
	if err != nil || raw == "" || hash == "" || hash == raw {
		t.Fatalf("token raw=%q hash=%q err=%v", raw, hash, err)
	}
	if HashToken(raw) != hash {
		t.Fatal("hash mismatch")
	}
	got := ResetLink("http://127.0.0.1:5173/", "", raw)
	want := "http://127.0.0.1:5173/reset-password?token=" + raw
	if got != want {
		t.Fatalf("link=%q want %q", got, want)
	}
	if ResetLink("", "http://127.0.0.1:5174", "abc") != "http://127.0.0.1:5174/reset-password?token=abc" {
		t.Fatal("origin fallback")
	}
}

func TestSendDisabledIncomplete(t *testing.T) {
	if err := Send(Settings{}, "a@b.c", "s", "b"); !errors.Is(err, ErrDisabled) {
		t.Fatalf("disabled: %v", err)
	}
	if err := Send(Settings{Enabled: true}, "a@b.c", "s", "b"); !errors.Is(err, ErrIncomplete) {
		t.Fatalf("incomplete: %v", err)
	}
}
