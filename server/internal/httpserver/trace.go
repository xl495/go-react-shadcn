package httpserver

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go-react-shadcn/internal/models"
)

const ctxTraceKey = "traceId"

func traceID(c *gin.Context) string {
	if v, ok := c.Get(ctxTraceKey); ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return ""
}

func (a *App) traceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := strings.TrimSpace(c.GetHeader("X-Trace-Id"))
		if _, err := uuid.Parse(id); err != nil {
			id = uuid.NewString()
		}
		c.Set(ctxTraceKey, id)
		c.Header("X-Trace-Id", id)
		start := time.Now()
		c.Next()
		slog.Info("http",
			"trace_id", id,
			"method", c.Request.Method,
			"path", c.FullPath(),
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
		)
	}
}

func peekBody(r io.ReadCloser, max int) (string, io.ReadCloser) {
	if r == nil {
		return "", r
	}
	limited := io.LimitReader(r, int64(max+1))
	raw, _ := io.ReadAll(limited)
	_ = r.Close()
	rest := io.NopCloser(bytes.NewReader(raw))
	return truncateText(string(raw), max), rest
}

func sensitiveAuthPath(path string) bool {
	switch path {
	case "/api/v1/auth/login", "/api/v1/auth/google", "/api/v1/auth/password",
		"/api/v1/auth/reset-password", "/api/v1/auth/forgot-password":
		return true
	default:
		return false
	}
}

var secretJSONKeys = map[string]struct{}{
	"password": {}, "oldpassword": {}, "newpassword": {}, "idtoken": {},
	"captchatoken": {}, "captchacode": {}, "token": {}, "secret": {},
	"clientsecret": {}, "authorization": {},
}

func redactRequestBody(path, raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	if redacted, ok := redactJSONSecrets(raw); ok {
		return redacted
	}
	if sensitiveAuthPath(path) {
		return "[redacted]"
	}
	return raw
}

func redactJSONSecrets(raw string) (string, bool) {
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return raw, false
	}
	redactJSONValue(v)
	out, err := json.Marshal(v)
	if err != nil {
		return "[redacted]", true
	}
	return string(out), true
}

func redactJSONValue(v any) {
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			if isSecretJSONKey(k) {
				if child != nil && child != "" {
					t[k] = "********"
				}
				continue
			}
			redactJSONValue(child)
		}
	case []any:
		for _, child := range t {
			redactJSONValue(child)
		}
	}
}

func isSecretJSONKey(k string) bool {
	n := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(k, "_", ""), "-", ""))
	_, ok := secretJSONKeys[n]
	return ok
}

func (a *App) logAPIRequests() gin.HandlerFunc {
	const maxBody = 4096
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		if path == "/metrics" || path == "/health" || strings.HasPrefix(path, "/uploads") {
			c.Next()
			return
		}

		var reqBody string
		if c.Request.Body != nil && c.Request.Method != http.MethodGet {
			raw, rest := peekBody(c.Request.Body, maxBody)
			c.Request.Body = rest
			reqBody = redactRequestBody(path, raw)
		}
		c.Next()

		username := ""
		if u := currentUser(c); u != nil {
			username = u.Username
		}
		errStack := ""
		if c.Writer.Status() >= 500 {
			if errs := c.Errors.String(); errs != "" {
				errStack = truncateText(errs, maxBody)
			}
		}
		a.apiLogs.enqueue(models.APILog{
			TraceID:     traceID(c),
			Username:    username,
			Method:      c.Request.Method,
			Path:        path,
			Status:      c.Writer.Status(),
			LatencyMs:   time.Since(start).Milliseconds(),
			RequestBody: reqBody,
			ErrorStack:  errStack,
		})
	}
}

func truncateText(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}

func (a *App) recordLoginLog(c *gin.Context, username, status, failReason string) {
	row := models.LoginLog{
		Username:   username,
		IP:         c.ClientIP(),
		UserAgent:  truncateText(c.GetHeader("User-Agent"), 512),
		Location:   a.lookupLocation(c.ClientIP()),
		Status:     status,
		FailReason: failReason,
	}
	if a.apiLogs != nil {
		a.apiLogs.enqueueLogin(row)
		return
	}
	_ = a.DB.Create(&row).Error
}

func (a *App) lookupLocation(ip string) string {
	if ip == "" || ip == "127.0.0.1" || ip == "::1" {
		return "local"
	}
	return ""
}

func (a *App) purgeOldLogs(retentionDays int) error {
	if retentionDays <= 0 {
		return nil
	}
	if a.apiLogs != nil {
		a.apiLogs.Flush()
	}
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	for _, model := range []any{&models.LoginLog{}, &models.OpLog{}, &models.APILog{}} {
		if err := a.DB.Where("created_at < ?", cutoff).Delete(model).Error; err != nil {
			return err
		}
	}
	return nil
}

func diffJSON(oldVal, newVal any) (string, string) {
	if oldVal == nil && newVal == nil {
		return "", ""
	}
	oldB, _ := json.Marshal(oldVal)
	newB, _ := json.Marshal(newVal)
	return string(oldB), string(newB)
}
