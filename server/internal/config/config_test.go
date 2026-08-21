package config

import "testing"

func TestValidateProductionSecrets(t *testing.T) {
	cfg := Config{
		DevMode:         false,
		JWTSecret:       "abcdefghijklmnopqrstuvwxyz012345",
		MailUnsubSecret: "zyxwvutsrqponmlkjihgfedcba543210",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	cfg.CaptchaDebug = true
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected captcha debug rejected in production")
	}
	cfg.CaptchaDebug = false
	cfg.MailUnsubSecret = cfg.JWTSecret
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected unsub secret to differ from jwt")
	}
	cfg.DevMode = true
	cfg.JWTSecret = defaultJWTSecret
	cfg.MailUnsubSecret = ""
	if err := cfg.Validate(); err != nil {
		t.Fatalf("dev should skip: %v", err)
	}
}

func TestUnsubSecretFallsBackInDev(t *testing.T) {
	cfg := Config{JWTSecret: "jwt", MailUnsubSecret: ""}
	if cfg.UnsubSecret() != "jwt" {
		t.Fatalf("got %q", cfg.UnsubSecret())
	}
	cfg.MailUnsubSecret = "unsub"
	if cfg.UnsubSecret() != "unsub" {
		t.Fatalf("got %q", cfg.UnsubSecret())
	}
}
