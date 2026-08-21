package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const defaultJWTSecret = "dev-secret-change-me"

type Config struct {
	DevMode          bool
	Port             string
	DatabasePath     string
	JWTSecret        string
	MailUnsubSecret  string
	MetricsToken     string
	TrustedProxies   []string
	JWTTTL           time.Duration
	CaptchaDebug     bool
	CORSOrigin       string
	UploadDir        string
	APILogEnabled    bool
	APILogSample     int
	SessionCache     time.Duration
	LogRetentionDays int
}

func Load() Config {
	ttl := 8 * time.Hour
	if v := os.Getenv("JWT_TTL"); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil {
			ttl = parsed
		}
	}
	dbPath := absPath(env("DATABASE_PATH", "./data/app.db"))
	upload := env("UPLOAD_DIR", "")
	if upload == "" {
		dir := filepath.Dir(dbPath)
		if dir == "." || dir == "" {
			dir = "data"
		}
		upload = filepath.Join(dir, "uploads")
	}
	upload = absPath(upload)
	dev := os.Getenv("APP_ENV") != "production" && os.Getenv("APP_ENV") != "prod"
	sample := 1
	if !dev {
		sample = 5
	}
	if v := os.Getenv("API_LOG_SAMPLE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			sample = n
		}
	}
	sessTTL := 30 * time.Second
	if v := os.Getenv("SESSION_CACHE_TTL"); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil {
			sessTTL = parsed
		}
	}
	retention := 30
	if v := os.Getenv("LOG_RETENTION_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			retention = n
		}
	}
	return Config{
		DevMode:          dev,
		Port:             env("PORT", "8080"),
		DatabasePath:     dbPath,
		JWTSecret:        env("JWT_SECRET", defaultJWTSecret),
		MailUnsubSecret:  strings.TrimSpace(os.Getenv("MAIL_UNSUB_SECRET")),
		MetricsToken:     strings.TrimSpace(os.Getenv("METRICS_TOKEN")),
		TrustedProxies:   splitCSV(os.Getenv("TRUSTED_PROXIES")),
		JWTTTL:           ttl,
		CaptchaDebug:     os.Getenv("CAPTCHA_DEBUG") == "1",
		CORSOrigin:       env("CORS_ORIGIN", corsDefault(dev)),
		UploadDir:        upload,
		APILogEnabled:    os.Getenv("API_LOGS") != "0",
		APILogSample:     sample,
		SessionCache:     sessTTL,
		LogRetentionDays: retention,
	}
}

func (c Config) UnsubSecret() string {
	if strings.TrimSpace(c.MailUnsubSecret) != "" {
		return c.MailUnsubSecret
	}
	return c.JWTSecret
}

func (c Config) Validate() error {
	if c.DevMode {
		return nil
	}
	if strings.TrimSpace(c.JWTSecret) == "" {
		return fmt.Errorf("JWT_SECRET is required in production")
	}
	if c.JWTSecret == defaultJWTSecret {
		return fmt.Errorf("JWT_SECRET must not use the default dev value in production")
	}
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters in production")
	}
	if strings.TrimSpace(c.MailUnsubSecret) == "" {
		return fmt.Errorf("MAIL_UNSUB_SECRET is required in production")
	}
	if c.MailUnsubSecret == c.JWTSecret {
		return fmt.Errorf("MAIL_UNSUB_SECRET must be distinct from JWT_SECRET")
	}
	if len(c.MailUnsubSecret) < 32 {
		return fmt.Errorf("MAIL_UNSUB_SECRET must be at least 32 characters in production")
	}
	if c.CaptchaDebug {
		return fmt.Errorf("CAPTCHA_DEBUG must not be enabled in production")
	}
	return nil
}

func corsDefault(dev bool) string {
	if !dev {
		return ""
	}
	return "http://localhost:5173,http://localhost:5174"
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitCSV(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func absPath(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	wd, err := os.Getwd()
	if err != nil {
		return p
	}
	return filepath.Clean(filepath.Join(wd, p))
}
