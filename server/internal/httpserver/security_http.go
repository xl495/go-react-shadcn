package httpserver

import (
	"crypto/subtle"
	"net"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	jsonBodyLimit      = int64(1 << 20)
	multipartBodyLimit = int64(3 << 20)
)

func (a *App) configureTrustedProxies(r *gin.Engine) {
	proxies := a.Cfg.TrustedProxies
	if len(proxies) == 0 {
		if a.Cfg.DevMode {
			_ = r.SetTrustedProxies(nil)
			return
		}
		proxies = []string{"127.0.0.1", "::1", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}
	}
	_ = r.SetTrustedProxies(proxies)
}

func (a *App) securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("X-XSS-Protection", "0")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'")
		if a != nil && !a.Cfg.DevMode {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		c.Next()
	}
}

func (a *App) limitRequestBody() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body == nil || c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead || c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}
		limit := jsonBodyLimit
		ct := c.ContentType()
		if strings.HasPrefix(ct, "multipart/") {
			limit = multipartBodyLimit
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Next()
	}
}

func (a *App) protectMetrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.metricsAllowed(c) {
			c.Next()
			return
		}
		c.AbortWithStatus(http.StatusUnauthorized)
	}
}

func (a *App) metricsAllowed(c *gin.Context) bool {
	token := strings.TrimSpace(a.Cfg.MetricsToken)
	if token != "" {
		got := strings.TrimSpace(c.GetHeader("X-Metrics-Token"))
		if got == "" && strings.HasPrefix(c.GetHeader("Authorization"), "Bearer ") {
			got = strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
		}
		return subtle.ConstantTimeCompare([]byte(got), []byte(token)) == 1
	}
	if a.Cfg.DevMode {
		return true
	}
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		host = c.Request.RemoteAddr
	}
	return host == "127.0.0.1" || host == "::1"
}

func (a *App) uploadHeaders() gin.HandlerFunc {
	allowed := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, ".svg": true,
	}
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		ext := strings.ToLower(filepath.Ext(c.Request.URL.Path))
		if ext == "" {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		if !allowed[ext] {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		if ext == ".svg" {
			c.Header("Content-Disposition", "attachment")
			c.Header("Content-Type", "image/svg+xml")
		} else {
			c.Header("Content-Disposition", "inline")
		}
		c.Next()
	}
}
