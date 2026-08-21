package httpserver

import (
	"compress/gzip"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type gzipWriter struct {
	gin.ResponseWriter
	writer *gzip.Writer
}

func (g gzipWriter) Write(data []byte) (int, error) {
	return g.writer.Write(data)
}

func (g gzipWriter) WriteString(s string) (int, error) {
	return g.writer.Write([]byte(s))
}

func (g gzipWriter) Flush() {
	_ = g.writer.Flush()
	if f, ok := g.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func gzipIfAsked() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "HEAD" || strings.HasPrefix(c.Request.URL.Path, "/uploads/") {
			c.Next()
			return
		}
		if !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}
		c.Header("Content-Encoding", "gzip")
		c.Header("Vary", "Accept-Encoding")
		gz := gzip.NewWriter(c.Writer)
		defer func() { _ = gz.Close() }()
		c.Writer = gzipWriter{ResponseWriter: c.Writer, writer: gz}
		c.Next()
	}
}
