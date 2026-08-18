package config

import (
	"os"
	"path/filepath"
	"time"
)

type Config struct {
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
	dbPath := env("DATABASE_PATH", "./data/app.db")
	upload := env("UPLOAD_DIR", "")
	if upload == "" {
		dir := filepath.Dir(dbPath)
		if dir == "." || dir == "" {
			dir = "data"
		}
		upload = filepath.Join(dir, "uploads")
	}
	return Config{
		Port:         env("PORT", "8080"),
		DatabasePath: dbPath,
		JWTSecret:    env("JWT_SECRET", "dev-secret-change-me"),
		JWTTTL:       ttl,
		CaptchaDebug: os.Getenv("CAPTCHA_DEBUG") == "1",
		CORSOrigin:   env("CORS_ORIGIN", "http://localhost:5173"),
		UploadDir:    upload,
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
