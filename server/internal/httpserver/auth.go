package httpserver

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/mailer"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/passwd"
	"go-react-shadcn/internal/security"
	"go-react-shadcn/internal/seed"
	"gorm.io/gorm"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Client   string `json:"client"`
	challengeInput
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

	if !a.requireCaptcha(c, req.challengeInput, "login") {
		if req.CaptchaID != "" || req.CaptchaToken != "" {
			a.recordLoginLog(c, req.Username, "failed", "invalid captcha")
		}
		return
	}

	if req.Username == "" || req.Password == "" {
		a.recordLoginLog(c, req.Username, "failed", "missing credentials")
		fail(c, http.StatusUnauthorized, 40103, "invalid credentials")
		return
	}

	kind := loginClientKind(req.Client)
	var user models.User
	if err := a.loadAccount(kind, &user, "username = ?", req.Username); err != nil {
		_ = passwd.Match(passwd.DummyHash(), req.Password)
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

	if !user.EmailVerified && user.Kind == models.UserKindWeb && user.GoogleID == "" {
		a.recordLoginLog(c, user.Username, "failed", "email unverified")
		fail(c, http.StatusForbidden, CodeEmailUnverified, "verify your email before signing in")
		return
	}

	if user.Status != "active" || !passwd.Match(user.PasswordHash, req.Password) {
		user.FailedLoginCount++
		updates := map[string]any{"failed_login_count": user.FailedLoginCount}
		if until := a.LoginGuard.LockedUntil(now, user.FailedLoginCount); until != nil {
			updates["locked_until"] = until
		}
		_ = a.updateAccount(&user, updates)
		a.recordLoginLog(c, user.Username, "failed", "invalid credentials")
		fail(c, http.StatusUnauthorized, 40103, "invalid credentials")
		return
	}

	if !a.Cfg.DevMode && seed.IsDefaultPassword(req.Password) {
		a.recordLoginLog(c, user.Username, "failed", "seed password")
		fail(c, http.StatusForbidden, CodeSeedPassword, "default seed password is disabled in production")
		return
	}

	a.finishLogin(c, user, ip, "")
}

func (a *App) finishLogin(c *gin.Context, user models.User, ip, successDetail string) {
	now := time.Now()
	if isAnomalousLogin(user, ip) {
		a.recordLoginLog(c, user.Username, "warning", "anomalous ip:"+ip+" prev:"+user.LastLoginIP)
	}

	user.LastLoginAt = &now
	user.LastLoginIP = ip
	user.FailedLoginCount = 0
	user.LockedUntil = nil
	_ = a.updateAccount(&user, map[string]any{
		"last_login_at":      user.LastLoginAt,
		"last_login_ip":      user.LastLoginIP,
		"failed_login_count": 0,
		"locked_until":       nil,
	})

	roles := roleCodes(user.Roles)
	tok, exp, err := a.Tokens.Issue(user.ID, user.Username, roles, user.TokenVersion, user.Kind)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50003, "failed to issue token")
		return
	}
	claims, err := a.Tokens.Parse(tok)
	if err != nil || claims.ID == "" {
		fail(c, http.StatusInternalServerError, 50003, "failed to issue token")
		return
	}
	row := models.AuthSession{
		UserID: user.ID, UserKind: models.NormalizeUserKind(user.Kind), JTI: claims.ID,
		IP: ip, UserAgent: c.GetHeader("User-Agent"), ExpiresAt: exp,
	}
	if err := a.DB.Create(&row).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50003, "failed to issue token")
		return
	}
	a.recordLoginLog(c, user.Username, "success", successDetail)
	ok(c, loginData{
		Token:     tok,
		ExpiresAt: exp,
		User:      a.toUserDTO(user),
	})
}

func (a *App) handleMe(c *gin.Context) {
	user, _, err := a.currentAccount(c)
	if err != nil {
		fail(c, http.StatusNotFound, 40401, "user not found")
		return
	}
	ok(c, a.toUserDTO(user))
}

type updateProfileRequest struct {
	Nickname       string `json:"nickname"`
	Email          string `json:"email"`
	Phone          string `json:"phone"`
	Gender         string `json:"gender"`
	Department     string `json:"department"`
	Title          string `json:"title"`
	Remark         string `json:"remark"`
	Timezone       string `json:"timezone"`
	MarketingOptIn *bool  `json:"marketingOptIn"`
}

type changePasswordRequest struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

func (a *App) handleUpdateProfile(c *gin.Context) {
	user, _, err := a.currentAccount(c)
	if err != nil {
		fail(c, http.StatusNotFound, 40401, "user not found")
		return
	}
	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, 40040, "invalid request body")
		return
	}
	if !a.requireDictValue(c, seed.DictGender, req.Gender) ||
		!a.requireDepartmentCode(c, req.Department) {
		return
	}
	if req.Email != "" && a.emailTaken(user.Kind, req.Email, user.ID) {
		fail(c, http.StatusConflict, CodeEmailExists, "email already exists")
		return
	}
	user.Nickname = req.Nickname
	user.Email = req.Email
	user.Phone = req.Phone
	user.Gender = req.Gender
	user.Department = req.Department
	if strings.TrimSpace(req.Department) == "" {
		user.DepartmentID = nil
	}
	a.applyDepartmentLink(&user)
	user.Title = req.Title
	user.Remark = req.Remark
	if req.Timezone != "" {
		tz, err := mailer.NormalizeTimezone(req.Timezone)
		if err != nil {
			fail(c, http.StatusBadRequest, CodeInvalidTimezone, "invalid timezone")
			return
		}
		user.Timezone = tz
	}
	if req.MarketingOptIn != nil {
		user.MarketingOptIn = *req.MarketingOptIn
	}
	if err := a.saveAccount(&user); err != nil {
		fail(c, http.StatusInternalServerError, 50041, "failed to update profile")
		return
	}
	ok(c, a.toUserDTO(user))
}

func (a *App) handleChangePassword(c *gin.Context) {
	user, _, err := a.currentAccount(c)
	if err != nil {
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
	if a.failIfWeakPassword(c, req.NewPassword, user.Username) {
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
	if err := a.updateAccount(&user, map[string]any{
		"password_hash": hash,
		"token_version": user.TokenVersion + 1,
	}); err != nil {
		fail(c, http.StatusInternalServerError, CodeChangePassword, "failed to change password")
		return
	}
	a.revokeAuthSessions(user.Kind, user.ID)
	a.sessions.invalidate(user.Kind, user.ID)
	ok(c, gin.H{"changed": true})
}

func (a *App) handleLogout(c *gin.Context) {
	claims := currentUser(c)
	if claims == nil {
		fail(c, http.StatusUnauthorized, CodeMissingToken, "missing bearer token")
		return
	}
	kind := claimsKind(claims)
	if claims.ID != "" {
		a.revokeAuthSessionJTI(claims.ID)
	} else if err := a.accounts(kind).Where("id = ?", claims.UserID).Update("token_version", gorm.Expr("token_version + 1")).Error; err != nil {
		fail(c, http.StatusInternalServerError, CodeChangePassword, "failed to logout")
		return
	}
	a.sessions.invalidate(kind, claims.UserID)
	ok(c, gin.H{"loggedOut": true})
}

func (a *App) failIfWeakPassword(c *gin.Context, plain, username string) bool {
	err := a.passwordIssue(plain, username)
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, passwd.ErrTooShort):
		fail(c, http.StatusBadRequest, CodePasswordTooShort, err.Error())
	case errors.Is(err, passwd.ErrSameAsUser):
		fail(c, http.StatusBadRequest, CodePasswordSameAsUser, err.Error())
	case errors.Is(err, passwd.ErrSeed):
		fail(c, http.StatusBadRequest, CodePasswordSeed, err.Error())
	default:
		fail(c, http.StatusBadRequest, CodePasswordWeak, err.Error())
	}
	return true
}
