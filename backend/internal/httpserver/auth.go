package httpserver

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/passwd"
	"go-react-shadcn/internal/security"
	"go-react-shadcn/internal/seed"
)

type loginRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	CaptchaID   string `json:"captchaId"`
	CaptchaCode string `json:"captchaCode"`
}

type loginData struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	User      userDTO   `json:"user"`
}

func (a *App) handleCaptcha(c *gin.Context) {
	ch, err := a.Captcha.Issue()
	if err != nil {
		fail(c, http.StatusInternalServerError, 50002, "failed to issue captcha")
		return
	}
	ok(c, ch)
}

func (a *App) handleLogin(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, 40001, "invalid request body")
		return
	}

	ip := c.ClientIP()
	if !a.LoginGuard.AllowIP(ip) {
		a.recordLoginLog(c, req.Username, "failed", "ip rate limited")
		fail(c, http.StatusTooManyRequests, 42901, "too many login attempts from this ip")
		return
	}

	if a.captchaEnabled() {
		if req.CaptchaID == "" || req.CaptchaCode == "" {
			fail(c, http.StatusBadRequest, 40002, "captcha required")
			return
		}
		if !a.Captcha.Verify(req.CaptchaID, req.CaptchaCode) {
			a.recordLoginLog(c, req.Username, "failed", "invalid captcha")
			fail(c, http.StatusBadRequest, 40003, "invalid captcha")
			return
		}
	}

	if req.Username == "" || req.Password == "" {
		a.recordLoginLog(c, req.Username, "failed", "missing credentials")
		fail(c, http.StatusUnauthorized, 40103, "invalid credentials")
		return
	}

	var user models.User
	if err := a.DB.Preload("Roles.Permissions").Where("username = ?", req.Username).First(&user).Error; err != nil {
		a.recordLoginLog(c, req.Username, "failed", "user not found")
		fail(c, http.StatusUnauthorized, 40103, "invalid credentials")
		return
	}

	now := time.Now()
	if security.IsLocked(user.LockedUntil, now) {
		a.recordLoginLog(c, user.Username, "failed", "account locked")
		fail(c, http.StatusForbidden, 40310, "account locked")
		return
	}

	if user.Status != "active" || !passwd.Match(user.PasswordHash, req.Password) {
		user.FailedLoginCount++
		updates := map[string]any{"failed_login_count": user.FailedLoginCount}
		if until := a.LoginGuard.LockedUntil(now, user.FailedLoginCount); until != nil {
			updates["locked_until"] = until
		}
		_ = a.DB.Model(&user).Updates(updates).Error
		a.recordLoginLog(c, user.Username, "failed", "invalid credentials")
		fail(c, http.StatusUnauthorized, 40103, "invalid credentials")
		return
	}

	if isAnomalousLogin(user, ip) {
		a.recordLoginLog(c, user.Username, "warning", "anomalous ip:"+ip+" prev:"+user.LastLoginIP)
	}

	user.LastLoginAt = &now
	user.LastLoginIP = ip
	user.FailedLoginCount = 0
	user.LockedUntil = nil
	_ = a.DB.Model(&user).Updates(map[string]any{
		"last_login_at":      user.LastLoginAt,
		"last_login_ip":      user.LastLoginIP,
		"failed_login_count": 0,
		"locked_until":       nil,
	}).Error

	roles := roleCodes(user.Roles)
	tok, exp, err := a.Tokens.Issue(user.ID, user.Username, roles, user.TokenVersion)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50003, "failed to issue token")
		return
	}
	a.recordLoginLog(c, user.Username, "success", "")
	ok(c, loginData{
		Token:     tok,
		ExpiresAt: exp,
		User:      toUserDTO(user),
	})
}

func (a *App) captchaEnabled() bool {
	var cfg models.SysConfig
	if err := a.DB.Where(`"key" = ?`, "app.captcha_enabled").First(&cfg).Error; err != nil {
		return true
	}
	return cfg.Value == "1" || cfg.Value == "true"
}

func (a *App) handleMe(c *gin.Context) {
	claims := currentUser(c)
	var user models.User
	if err := a.DB.Preload("Roles.Permissions").First(&user, claims.UserID).Error; err != nil {
		fail(c, http.StatusNotFound, 40401, "user not found")
		return
	}
	ok(c, toUserDTO(user))
}

type updateProfileRequest struct {
	Nickname   string `json:"nickname"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	Gender     string `json:"gender"`
	Department string `json:"department"`
	Title      string `json:"title"`
	Remark     string `json:"remark"`
}

type changePasswordRequest struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

func (a *App) handleUpdateProfile(c *gin.Context) {
	claims := currentUser(c)
	var user models.User
	if err := a.DB.Preload("Roles.Permissions").First(&user, claims.UserID).Error; err != nil {
		fail(c, http.StatusNotFound, 40401, "user not found")
		return
	}
	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, 40040, "invalid request body")
		return
	}
	if !a.requireDictValue(c, seed.DictGender, req.Gender) ||
		!a.requireDictValue(c, seed.DictDepartment, req.Department) {
		return
	}
	user.Nickname = req.Nickname
	user.Email = req.Email
	user.Phone = req.Phone
	user.Gender = req.Gender
	user.Department = req.Department
	a.applyDepartmentLink(&user)
	user.Title = req.Title
	user.Remark = req.Remark
	if err := a.DB.Save(&user).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50041, "failed to update profile")
		return
	}
	ok(c, toUserDTO(user))
}

func (a *App) handleChangePassword(c *gin.Context) {
	claims := currentUser(c)
	var user models.User
	if err := a.DB.First(&user, claims.UserID).Error; err != nil {
		fail(c, http.StatusNotFound, 40401, "user not found")
		return
	}
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, 40040, "invalid request body")
		return
	}
	if req.OldPassword == "" || req.NewPassword == "" {
		fail(c, http.StatusBadRequest, 40042, "old and new password required")
		return
	}
	if len(req.NewPassword) < 8 {
		fail(c, http.StatusBadRequest, 40043, "password must be at least 8 characters")
		return
	}
	if !passwd.Match(user.PasswordHash, req.OldPassword) {
		fail(c, http.StatusBadRequest, 40041, "current password is wrong")
		return
	}
	hash, err := passwd.Hash(req.NewPassword)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50011, "failed to hash password")
		return
	}
	if err := a.DB.Model(&user).Updates(map[string]any{
		"password_hash": hash,
		"token_version": user.TokenVersion + 1,
	}).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50042, "failed to change password")
		return
	}
	ok(c, gin.H{"changed": true})
}
