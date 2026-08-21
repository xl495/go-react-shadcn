package httpserver

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/migrate"
)

var latencyBuckets = []float64{5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000}

type hist struct {
	count uint64
	sum   float64
	obs   []uint64
}

type httpMetrics struct {
	mu      sync.Mutex
	counts  map[string]uint64
	latency hist
}

func newHTTPMetrics() *httpMetrics {
	return &httpMetrics{
		counts:  make(map[string]uint64),
		latency: hist{obs: make([]uint64, len(latencyBuckets)+1)},
	}
}

func (m *httpMetrics) inc(method, path string, status int, latencyMs int64) {
	key := fmt.Sprintf(`latch_http_requests_total{method=%q,path=%q,status="%d"}`, method, path, status)
	m.mu.Lock()
	m.counts[key]++
	ms := float64(latencyMs)
	m.latency.count++
	m.latency.sum += ms
	placed := false
	for i, le := range latencyBuckets {
		if ms <= le {
			m.latency.obs[i]++
			placed = true
			break
		}
	}
	if !placed {
		m.latency.obs[len(latencyBuckets)]++
	}
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
	b.WriteString("# HELP latch_http_requests_total HTTP requests handled by gra.\n")
	b.WriteString("# TYPE latch_http_requests_total counter\n")
	for _, k := range keys {
		fmt.Fprintf(&b, "%s %d\n", k, m.counts[k])
	}
	b.WriteString("# HELP latch_http_request_duration_ms Request latency in milliseconds.\n")
	b.WriteString("# TYPE latch_http_request_duration_ms histogram\n")
	var cum uint64
	for i, le := range latencyBuckets {
		cum += m.latency.obs[i]
		fmt.Fprintf(&b, "latch_http_request_duration_ms_bucket{le=\"%g\"} %d\n", le, cum)
	}
	cum += m.latency.obs[len(latencyBuckets)]
	fmt.Fprintf(&b, "latch_http_request_duration_ms_bucket{le=\"+Inf\"} %d\n", cum)
	fmt.Fprintf(&b, "latch_http_request_duration_ms_sum %g\n", m.latency.sum)
	fmt.Fprintf(&b, "latch_http_request_duration_ms_count %d\n", m.latency.count)
	return b.String()
}

func (a *App) observeRequests() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		if path == "/metrics" {
			return
		}
		a.metrics.inc(c.Request.Method, path, c.Writer.Status(), time.Since(start).Milliseconds())
	}
}

func (a *App) handleLive(c *gin.Context) {
	ok(c, gin.H{"status": "live"})
}

func (a *App) dbPingOK() bool {
	sqlDB, err := a.DB.DB()
	if err != nil {
		return false
	}
	return sqlDB.Ping() == nil
}

func (a *App) handleReady(c *gin.Context) {
	if !a.dbPingOK() {
		fail(c, http.StatusServiceUnavailable, CodeUnhealthy, "unhealthy")
		return
	}
	out := gin.H{"status": "ready"}
	version, dirty, err := migrate.Version(a.Cfg.DatabasePath)
	if err == nil && dirty {
		fail(c, http.StatusServiceUnavailable, CodeUnhealthy, "migrate dirty")
		return
	}
	if err == nil {
		out["migrateVersion"] = version
	}
	ok(c, out)
}

func (a *App) handleHealth(c *gin.Context) {
	if !a.dbPingOK() {
		fail(c, http.StatusServiceUnavailable, CodeUnhealthy, "unhealthy")
		return
	}
	ok(c, gin.H{"status": "ok"})
}

func (a *App) handleMetrics(c *gin.Context) {
	c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	c.String(http.StatusOK, a.metrics.render()+a.auditDropMetrics())
}

func (a *App) auditDropMetrics() string {
	n := uint64(0)
	if a != nil && a.apiLogs != nil {
		n = a.apiLogs.droppedCount()
	}
	return "# HELP latch_audit_dropped_total Audit queue overflows that fell back to a synchronous write.\n" +
		"# TYPE latch_audit_dropped_total counter\n" +
		fmt.Sprintf("latch_audit_dropped_total %d\n", n)
}
