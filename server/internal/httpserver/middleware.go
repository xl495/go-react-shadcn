package httpserver

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/token"
)

const ctxUserKey = "authUser"

func (a *App) requireJWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			fail(c, http.StatusUnauthorized, 40101, "missing bearer token")
			return
		}
		raw := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		claims, err := a.Tokens.Parse(raw)
		if err != nil {
			fail(c, http.StatusUnauthorized, 40102, "invalid or expired token")
			return
		}
		tokenVersion, status := 0, ""
		if snap, ok := a.sessions.get(claims.UserID); ok {
			tokenVersion, status = snap.tokenVersion, snap.status
		} else {
			var user models.User
			if err := a.DB.Select("id", "token_version", "status").First(&user, claims.UserID).Error; err != nil {
				fail(c, http.StatusUnauthorized, CodeInvalidToken, "invalid or expired token")
				return
			}
			tokenVersion, status = user.TokenVersion, user.Status
			a.sessions.put(user.ID, tokenVersion, status)
		}
		if status != "active" || tokenVersion != claims.TokenVersion {
			fail(c, http.StatusUnauthorized, CodeInvalidToken, "invalid or expired token")
			return
		}
		c.Set(ctxUserKey, claims)
		c.Next()
	}
}

func currentUser(c *gin.Context) *token.Claims {
	v, ok := c.Get(ctxUserKey)
	if !ok {
		return nil
	}
	claims, _ := v.(*token.Claims)
	return claims
}

func (a *App) requireCasbin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := currentUser(c)
		if user == nil {
			fail(c, http.StatusUnauthorized, 40101, "missing bearer token")
			return
		}
		allowed, err := a.Enforcer.Enforce(user.Username, c.FullPath(), c.Request.Method)
		if err != nil {
			fail(c, http.StatusInternalServerError, 50001, "permission check failed")
			return
		}
		if !allowed {
			fail(c, http.StatusForbidden, 40301, "permission denied")
			return
		}
		c.Next()
	}
}
