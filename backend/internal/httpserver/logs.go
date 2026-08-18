package httpserver

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/models"
)

func (a *App) recordLog(c *gin.Context, module, action, detail string) {
	username := ""
	if u := currentUser(c); u != nil {
		username = u.Username
	}
	path := c.FullPath()
	if path == "" {
		path = c.Request.URL.Path
	}
	_ = a.DB.Create(&models.OpLog{
		Username: username,
		Module:   module,
		Action:   action,
		Method:   c.Request.Method,
		Path:     path,
		Status:   c.Writer.Status(),
		IP:       c.ClientIP(),
		Detail:   detail,
	}).Error
}

func (a *App) logMutations() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodOptions {
			return
		}
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		if strings.HasPrefix(path, "/api/v1/logs") {
			return
		}
		username := ""
		if u := currentUser(c); u != nil {
			username = u.Username
		}
		module := moduleOf(path)
		_ = a.DB.Create(&models.OpLog{
			Username:  username,
			Module:    module,
			Action:    c.Request.Method,
			Method:    c.Request.Method,
			Path:      path,
			Status:    c.Writer.Status(),
			IP:        c.ClientIP(),
			LatencyMs: time.Since(start).Milliseconds(),
			Detail:    c.Request.URL.RawQuery,
		}).Error
	}
}

func moduleOf(path string) string {
	switch {
	case strings.Contains(path, "/users"):
		return "user"
	case strings.Contains(path, "/roles"):
		return "role"
	case strings.Contains(path, "/permissions"):
		return "perm"
	case strings.Contains(path, "/dicts") || strings.Contains(path, "/dict-items"):
		return "dict"
	case strings.Contains(path, "/configs"):
		return "config"
	case strings.Contains(path, "/auth"):
		return "auth"
	default:
		return "system"
	}
}

func (a *App) handleListLogs(c *gin.Context) {
	q := a.DB.Model(&models.OpLog{}).Order("id desc")
	if u := c.Query("username"); u != "" {
		q = q.Where("username = ?", u)
	}
	if m := c.Query("module"); m != "" {
		q = q.Where("module = ?", m)
	}
	limit := 100
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	var rows []models.OpLog
	if err := q.Limit(limit).Find(&rows).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50080, "failed to list logs")
		return
	}
	ok(c, rows)
}

func (a *App) handleClearLogs(c *gin.Context) {
	if err := a.DB.Where("1 = 1").Delete(&models.OpLog{}).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50081, "failed to clear logs")
		return
	}
	ok(c, gin.H{"cleared": true})
}
