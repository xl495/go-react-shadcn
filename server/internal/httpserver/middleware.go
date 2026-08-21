package httpserver

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/seed"
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
		kind := claimsKind(claims)
		tokenVersion, status := 0, ""
		if snap, ok := a.sessions.get(kind, claims.UserID); ok {
			tokenVersion, status = snap.tokenVersion, snap.status
		} else {
			var user models.User
			if err := a.accounts(kind).Select("id", "token_version", "status").First(&user, claims.UserID).Error; err != nil {
				fail(c, http.StatusUnauthorized, CodeInvalidToken, "invalid or expired token")
				return
			}
			tokenVersion, status = user.TokenVersion, user.Status
			a.sessions.put(kind, user.ID, tokenVersion, status)
		}
		if status != "active" || tokenVersion != claims.TokenVersion {
			fail(c, http.StatusUnauthorized, CodeInvalidToken, "invalid or expired token")
			return
		}
		if claims.ID != "" {
			var sess models.AuthSession
			err := a.DB.Select("id", "revoked_at", "expires_at").Where("jti = ?", claims.ID).First(&sess).Error
			if err != nil || sess.RevokedAt != nil || time.Now().After(sess.ExpiresAt) {
				fail(c, http.StatusUnauthorized, CodeInvalidToken, "invalid or expired token")
				return
			}
		}
		c.Set(ctxUserKey, claims)
		c.Next()
	}
}

func (a *App) casbinAllowed(sub, path, method string) (bool, error) {
	allowed, err := a.Enforcer.Enforce(sub, path, method)
	if err != nil || allowed {
		return allowed, err
	}
	// Saving the settings form uses PUT /configs/batch; treat it as config:update.
	if method == http.MethodPut && path == "/api/v1/configs/batch" {
		return a.Enforcer.Enforce(sub, "/api/v1/configs/:id", http.MethodPut)
	}
	return false, nil
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
		allowed, err := a.casbinAllowed(seed.CasbinSub(claimsKind(user), user.UserID), c.FullPath(), c.Request.Method)
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
