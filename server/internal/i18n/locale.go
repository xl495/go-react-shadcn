package i18n

import (
	"net/http"
	"strings"
)

const (
	ZhCN = "zh-CN"
	En   = "en"
)

func FromRequest(r *http.Request) string {
	if r == nil {
		return En
	}
	if v := strings.TrimSpace(r.Header.Get("X-Locale")); v != "" {
		return Normalize(v)
	}
	return Normalize(r.Header.Get("Accept-Language"))
}

func Normalize(raw string) string {
	tag := strings.TrimSpace(strings.Split(raw, ",")[0])
	tag = strings.TrimSpace(strings.Split(tag, ";")[0])
	if tag == "" {
		return En
	}
	lower := strings.ToLower(strings.ReplaceAll(tag, "_", "-"))
	if strings.HasPrefix(lower, "zh") {
		return ZhCN
	}
	if strings.HasPrefix(lower, "en") {
		return En
	}
	return En
}

func Error(locale string, code int, fallback string) string {
	if locale == ZhCN {
		if msg, ok := errorsZH[code]; ok {
			return msg
		}
	}
	if fallback != "" {
		return fallback
	}
	if msg, ok := errorsEN[code]; ok {
		return msg
	}
	return "request failed"
}
