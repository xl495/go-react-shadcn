package httpserver

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-react-shadcn/internal/models"
)

func (a *App) httpClient() *http.Client {
	if a.HTTP != nil {
		return a.HTTP
	}
	return &http.Client{Timeout: 8 * time.Second}
}

func (a *App) sysValue(key string) string {
	var cfg models.SysConfig
	if err := a.DB.Where(`"key" = ?`, key).First(&cfg).Error; err != nil {
		return ""
	}
	return strings.TrimSpace(cfg.Value)
}

func configOn(value string) bool {
	v := strings.ToLower(strings.TrimSpace(value))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func (a *App) sysOn(key string, fallback bool) bool {
	raw := a.sysValue(key)
	if raw == "" {
		return fallback
	}
	return configOn(raw)
}

func (a *App) captchaProvider() string {
	p := strings.ToLower(a.sysValue("auth.captcha_provider"))
	switch p {
	case "none", "image", "recaptcha", "turnstile":
		return p
	}
	if a.sysOn("app.captcha_enabled", true) {
		return "image"
	}
	return "none"
}

func (a *App) recaptchaMinScore() float64 {
	raw := a.sysValue("auth.recaptcha_min_score")
	if raw == "" {
		return 0.5
	}
	n, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0.5
	}
	if n < 0 {
		return 0
	}
	if n > 1 {
		return 1
	}
	return n
}

func (a *App) googleClientID() string {
	return a.sysValue("auth.google_client_id")
}

func (a *App) googleEnabled() bool {
	return a.sysOn("auth.google_enabled", false) && a.googleClientID() != ""
}

func (a *App) googleRegisterEnabled() bool {
	return a.googleEnabled() && a.sysOn("auth.google_register_enabled", false)
}
