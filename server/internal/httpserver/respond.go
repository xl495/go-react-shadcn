package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/i18n"
)

// Envelope is { code, message, data }. code is 0 on success and 1 on error.
// errorCode keeps the machine-readable application code for clients that need it.
type body struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data"`
	ErrorCode int    `json:"errorCode,omitempty"`
}

func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, body{Code: CodeOK, Message: "ok", Data: data})
}

func fail(c *gin.Context, status, errorCode int, message string) {
	msg := i18n.Error(i18n.FromRequest(c.Request), errorCode, message)
	c.AbortWithStatusJSON(status, body{Code: CodeFail, Message: msg, Data: nil, ErrorCode: errorCode})
}

func notFoundJSON(c *gin.Context) {
	fail(c, http.StatusNotFound, CodeRouteNotFound, "not found")
}

func methodNotAllowedJSON(c *gin.Context) {
	fail(c, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
}
