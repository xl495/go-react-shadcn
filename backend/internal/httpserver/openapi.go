package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/openapi"
)

func (a *App) handleOpenAPI(c *gin.Context) {
	c.Data(http.StatusOK, "application/yaml", openapi.Spec)
}
