package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const defaultJWTSecret = "dev-secret-change-me"

type Config struct {
	DevMode      bool
	Port         string
	DatabasePath string
	JWTSecret    string
	JWTTTL       time.Duration
	CaptchaDebug bool
	CORSOrigin   string
	UploadDir    string
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
	return Config{
		DevMode:      dev,
		Port:         env("PORT", "8080"),
		DatabasePath: dbPath,
		JWTSecret:    env("JWT_SECRET", defaultJWTSecret),
		JWTTTL:       ttl,
		CaptchaDebug: os.Getenv("CAPTCHA_DEBUG") == "1",
		CORSOrigin:   env("CORS_ORIGIN", "http://localhost:5173"),
		UploadDir:    upload,
	}
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
	return nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
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
