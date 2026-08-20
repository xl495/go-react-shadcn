package httpserver

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/models"
)

func (a *App) recordOpLog(c *gin.Context, module, action, description, detail string, oldVal, newVal any) {
	username := ""
	if u := currentUser(c); u != nil {
		username = u.Username
	}
	path := c.FullPath()
	if path == "" {
		path = c.Request.URL.Path
	}
	oldJSON, newJSON := diffJSON(oldVal, newVal)
	_ = a.DB.Create(&models.OpLog{
		TraceID:     traceID(c),
		Username:    username,
		Module:      module,
		Action:      action,
		Method:      c.Request.Method,
		Path:        path,
		Status:      c.Writer.Status(),
		IP:          c.ClientIP(),
		Detail:      detail,
		Description: description,
		OldValue:    oldJSON,
		NewValue:    newJSON,
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
		action := opAction(c.Request.Method)
		_ = a.DB.Create(&models.OpLog{
			TraceID:     traceID(c),
			Username:    username,
			Module:      module,
			Action:      action,
			Method:      c.Request.Method,
			Path:        path,
			Status:      c.Writer.Status(),
			IP:          c.ClientIP(),
			LatencyMs:   time.Since(start).Milliseconds(),
			Detail:      c.Request.URL.RawQuery,
			Description: c.Request.Method + " " + path,
		}).Error
	}
}

func opAction(method string) string {
	switch method {
	case http.MethodPost:
		return "create"
	case http.MethodPut, http.MethodPatch:
		return "update"
	case http.MethodDelete:
		return "delete"
	default:
		return strings.ToLower(method)
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
	case strings.Contains(path, "/departments"):
		return "dept"
	case strings.Contains(path, "/auth"):
		return "auth"
	default:
		return "system"
	}
}

func (a *App) handleListLoginLogs(c *gin.Context) {
	p := parsePage(c, 20, 200)
	q := a.DB.Model(&models.LoginLog{})
	if u := c.Query("username"); u != "" {
		q = q.Where("username = ?", u)
	}
	if s := c.Query("status"); s != "" {
		q = q.Where("status = ?", s)
	}
	var total int64
	_ = q.Count(&total).Error
	var rows []models.LoginLog
	if err := q.Order("id desc").Offset(p.Offset()).Limit(p.PageSize).Find(&rows).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50082, "failed to list login logs")
		return
	}
	ok(c, pageResult[models.LoginLog]{Items: rows, Total: total, Page: p.Page, PageSize: p.PageSize})
}

func (a *App) handleListOpLogs(c *gin.Context) {
	p := parsePage(c, 20, 200)
	q := a.DB.Model(&models.OpLog{})
	if u := c.Query("username"); u != "" {
		q = q.Where("username = ?", u)
	}
	if m := c.Query("module"); m != "" {
		q = q.Where("module = ?", m)
	}
	if act := c.Query("action"); act != "" {
		q = q.Where("action = ?", act)
	}
	var total int64
	_ = q.Count(&total).Error
	var rows []models.OpLog
	if err := q.Order("id desc").Offset(p.Offset()).Limit(p.PageSize).Find(&rows).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50080, "failed to list op logs")
		return
	}
	ok(c, pageResult[models.OpLog]{Items: rows, Total: total, Page: p.Page, PageSize: p.PageSize})
}

func (a *App) handleListAPILogs(c *gin.Context) {
	p := parsePage(c, 20, 200)
	q := a.DB.Model(&models.APILog{})
	if tid := c.Query("traceId"); tid != "" {
		q = q.Where("trace_id = ?", tid)
	}
	if path := c.Query("path"); path != "" {
		q = q.Where("path LIKE ?", "%"+path+"%")
	}
	var total int64
	_ = q.Count(&total).Error
	var rows []models.APILog
	if err := q.Order("id desc").Offset(p.Offset()).Limit(p.PageSize).Find(&rows).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50083, "failed to list api logs")
		return
	}
	ok(c, pageResult[models.APILog]{Items: rows, Total: total, Page: p.Page, PageSize: p.PageSize})
}

func (a *App) handleListLogs(c *gin.Context) {
	a.handleListOpLogs(c)
}

func (a *App) handleClearLogs(c *gin.Context) {
	kind := c.DefaultQuery("kind", "op")
	switch kind {
	case "login":
		if err := a.DB.Where("1 = 1").Delete(&models.LoginLog{}).Error; err != nil {
			fail(c, http.StatusInternalServerError, 50081, "failed to clear logs")
			return
		}
	case "api":
		if err := a.DB.Where("1 = 1").Delete(&models.APILog{}).Error; err != nil {
			fail(c, http.StatusInternalServerError, 50081, "failed to clear logs")
			return
		}
	default:
		if err := a.DB.Where("1 = 1").Delete(&models.OpLog{}).Error; err != nil {
			fail(c, http.StatusInternalServerError, 50081, "failed to clear logs")
			return
		}
	}
	ok(c, gin.H{"cleared": true})
}

func (a *App) handlePurgeLogs(c *gin.Context) {
	days := 30
	if v := c.Query("days"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			days = n
		}
	}
	if err := a.purgeOldLogs(days); err != nil {
		fail(c, http.StatusInternalServerError, 50084, "failed to purge logs")
		return
	}
	ok(c, gin.H{"purged": true, "retentionDays": days})
}

func (a *App) countLogs(username, module, action string) int64 {
	q := a.DB.Model(&models.OpLog{})
	if username != "" {
		q = q.Where("username = ?", username)
	}
	if module != "" {
		q = q.Where("module = ?", module)
	}
	if action != "" {
		q = q.Where("action = ?", action)
	}
	var n int64
	_ = q.Count(&n).Error
	return n
}
