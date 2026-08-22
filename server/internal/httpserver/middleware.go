package httpserver

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/security"
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
		mustChange := false
		var lockedUntil *time.Time
		if snap, ok := a.sessions.get(kind, claims.UserID); ok {
			tokenVersion, status, mustChange, lockedUntil = snap.tokenVersion, snap.status, snap.mustChange, snap.lockedUntil
		} else {
			var user models.User
			if err := a.accounts(kind).Select("id", "token_version", "status", "must_change_password", "locked_until").First(&user, claims.UserID).Error; err != nil {
				fail(c, http.StatusUnauthorized, CodeInvalidToken, "invalid or expired token")
				return
			}
			tokenVersion, status, mustChange, lockedUntil = user.TokenVersion, user.Status, user.MustChangePassword, user.LockedUntil
			a.sessions.put(kind, user.ID, tokenVersion, status, mustChange, lockedUntil)
		}
		if status != "active" || tokenVersion != claims.TokenVersion {
			fail(c, http.StatusUnauthorized, CodeInvalidToken, "invalid or expired token")
			return
		}
		if security.IsLocked(lockedUntil, time.Now()) {
			fail(c, http.StatusForbidden, CodeAccountLocked, "account locked")
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
		if kind == models.UserKindWeb && a.sysOn("app.maintenance", false) {
			if c.FullPath() != "/api/v1/auth/logout" {
				fail(c, http.StatusServiceUnavailable, CodeMaintenance, "the site is under maintenance")
				return
			}
		}
		c.Set(ctxUserKey, claims)
		if mustChange && !mustChangeAllowed(c) {
			fail(c, http.StatusForbidden, CodeMustChangePassword, "you must change your password")
			return
		}
		c.Next()
	}
}

func (a *App) optionalJWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.Next()
			return
		}
		raw := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		claims, err := a.Tokens.Parse(raw)
		if err != nil {
			c.Next()
			return
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
	if method == http.MethodPut && path == "/api/v1/nav-menus/reorder" {
		return a.Enforcer.Enforce(sub, "/api/v1/nav-menus/:id", http.MethodPut)
	}
	if method == http.MethodPost && path == "/api/v1/users/:id/reset-password" {
		return a.Enforcer.Enforce(sub, "/api/v1/users/:id", http.MethodPut)
	}
	if method == http.MethodPost && path == "/api/v1/users/:id/unlock" {
		return a.Enforcer.Enforce(sub, "/api/v1/users/:id", http.MethodPut)
	}
	if method == http.MethodPut && path == "/api/v1/users/batch-status" {
		return a.Enforcer.Enforce(sub, "/api/v1/users/:id", http.MethodPut)
	}
	if method == http.MethodPost && path == "/api/v1/roles/:id/copy" {
		return a.Enforcer.Enforce(sub, "/api/v1/roles", http.MethodPost)
	}
	if method == http.MethodGet && path == "/api/v1/online-sessions" {
		return a.Enforcer.Enforce(sub, "/api/v1/users/:id/sessions", http.MethodGet)
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
