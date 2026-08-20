package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/i18n"
)

type body struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, body{Code: 0, Message: "ok", Data: data})
}

func fail(c *gin.Context, status, code int, message string) {
	msg := i18n.Error(i18n.FromRequest(c.Request), code, message)
	c.AbortWithStatusJSON(status, body{Code: code, Message: msg})
}
