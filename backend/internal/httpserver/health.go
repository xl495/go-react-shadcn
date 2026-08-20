package httpserver

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

type httpMetrics struct {
	mu     sync.Mutex
	counts map[string]uint64
}

func newHTTPMetrics() *httpMetrics {
	return &httpMetrics{counts: make(map[string]uint64)}
}

func (m *httpMetrics) inc(method, path string, status int) {
	key := fmt.Sprintf(`latch_http_requests_total{method=%q,path=%q,status="%d"}`, method, path, status)
	m.mu.Lock()
	m.counts[key]++
	m.mu.Unlock()
}

func (m *httpMetrics) render() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	keys := make([]string, 0, len(m.counts))
	for k := range m.counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("# HELP latch_http_requests_total HTTP requests handled by Latch.\n")
	b.WriteString("# TYPE latch_http_requests_total counter\n")
	for _, k := range keys {
		fmt.Fprintf(&b, "%s %d\n", k, m.counts[k])
	}
	return b.String()
}

func (a *App) observeRequests() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		if path == "/metrics" {
			return
		}
		a.metrics.inc(c.Request.Method, path, c.Writer.Status())
	}
}

func (a *App) handleHealth(c *gin.Context) {
	sqlDB, err := a.DB.DB()
	if err != nil {
		fail(c, http.StatusServiceUnavailable, 50301, "unhealthy")
		return
	}
	if err := sqlDB.Ping(); err != nil {
		fail(c, http.StatusServiceUnavailable, 50301, "unhealthy")
		return
	}
	ok(c, gin.H{"status": "ok"})
}

func (a *App) handleMetrics(c *gin.Context) {
	c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	c.String(http.StatusOK, a.metrics.render())
}
