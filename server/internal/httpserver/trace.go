package httpserver

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go-react-shadcn/internal/models"
)

const ctxTraceKey = "traceId"

type responseCapture struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseCapture) Write(b []byte) (int, error) {
	if len(b) > 0 {
		_, _ = w.body.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

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
		id := c.GetHeader("X-Trace-Id")
		if id == "" {
			id = uuid.NewString()
		}
		c.Set(ctxTraceKey, id)
		c.Header("X-Trace-Id", id)
		c.Next()
	}
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
			raw, _ := io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewReader(raw))
			reqBody = truncateText(string(raw), maxBody)
		}

		capture := &responseCapture{ResponseWriter: c.Writer, body: &bytes.Buffer{}}
		c.Writer = capture
		c.Next()

		username := ""
		if u := currentUser(c); u != nil {
			username = u.Username
		}
		respBody := truncateText(capture.body.String(), maxBody)
		errStack := ""
		if c.Writer.Status() >= 500 {
			if errs := c.Errors.String(); errs != "" {
				errStack = truncateText(errs, maxBody)
			}
		}
		_ = a.DB.Create(&models.APILog{
			TraceID:      traceID(c),
			Username:     username,
			Method:       c.Request.Method,
			Path:         path,
			Status:       c.Writer.Status(),
			LatencyMs:    time.Since(start).Milliseconds(),
			RequestBody:  reqBody,
			ResponseBody: respBody,
			ErrorStack:   errStack,
		}).Error
	}
}

func truncateText(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}

func (a *App) recordLoginLog(c *gin.Context, username, status, failReason string) {
	_ = a.DB.Create(&models.LoginLog{
		Username:   username,
		IP:         c.ClientIP(),
		UserAgent:  truncateText(c.GetHeader("User-Agent"), 512),
		Location:   a.lookupLocation(c.ClientIP()),
		Status:     status,
		FailReason: failReason,
	}).Error
}

func (a *App) lookupLocation(ip string) string {
	if ip == "" || ip == "127.0.0.1" || ip == "::1" {
		return "local"
	}
	return "unknown"
}

func (a *App) purgeOldLogs(retentionDays int) error {
	if retentionDays <= 0 {
		return nil
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
