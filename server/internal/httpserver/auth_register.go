package httpserver

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/i18n"
	"go-react-shadcn/internal/mailer"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/passwd"
	"go-react-shadcn/internal/seed"
	"gorm.io/gorm"
)

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Nickname string `json:"nickname"`
	Client   string `json:"client"`
	challengeInput
}

func (a *App) handleRegister(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, CodeInvalidBody, "invalid request body")
		return
	}
	if !a.sysOn("auth.register_enabled", true) {
		fail(c, http.StatusForbidden, CodeRegisterDisabled, "registration is disabled")
		return
	}
	if a.rejectWebMaintenance(c, req.Client) {
		return
	}
	if loginClientKind(req.Client) != models.UserKindWeb {
		fail(c, http.StatusForbidden, CodeWrongClient, "email registration is only available on the web client")
		return
	}
	ip := c.ClientIP()
	if a.LoginGuard != nil && !a.LoginGuard.AllowIP(ip) {
		fail(c, http.StatusTooManyRequests, CodeLoginRateLimited, "too many login attempts from this ip")
		return
	}
	if !a.requireCaptcha(c, req.challengeInput, "register") {
		return
	}
	username := strings.TrimSpace(req.Username)
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if username == "" || email == "" || req.Password == "" {
		fail(c, http.StatusBadRequest, CodeUserPassRequired, "username, email and password required")
		return
	}
	if a.failIfWeakPassword(c, req.Password, username) {
		return
	}
	kind := models.UserKindWeb
	var n int64
	if err := a.accounts(kind).Where("username = ?", username).Count(&n).Error; err != nil {
		fail(c, http.StatusInternalServerError, CodeAssignRoles, "failed to create user")
		return
	}
	if n > 0 {
		fail(c, http.StatusConflict, CodeUserExists, "username already exists")
		return
	}
	if err := a.accounts(kind).Where("lower(email) = ?", email).Count(&n).Error; err != nil {
		fail(c, http.StatusInternalServerError, CodeAssignRoles, "failed to create user")
		return
	}
	if n > 0 {
		fail(c, http.StatusConflict, CodeEmailExists, "email already exists")
		return
	}
	hash, err := passwd.Hash(req.Password)
	if err != nil {
		fail(c, http.StatusInternalServerError, CodeHashPassword, "failed to hash password")
		return
	}
	roles, err := a.defaultRolesForKind(kind, nil)
	if err != nil {
		fail(c, http.StatusInternalServerError, CodeAssignRoles, "failed to create user")
		return
	}
	nick := strings.TrimSpace(req.Nickname)
	if nick == "" {
		nick = username
	}
	user := models.User{
		Username:       username,
		PasswordHash:   hash,
		Email:          email,
		Nickname:       nick,
		Status:         "active",
		Timezone:       mailer.DefaultTimezone,
		MarketingOptIn: false,
		Kind:           kind,
	}
	if err := a.withTx(func(tx *gorm.DB) error {
		if err := models.Accounts(tx, kind).Create(&user).Error; err != nil {
			return err
		}
		return models.ReplaceUserRoles(tx, kind, user.ID, roles)
	}); err != nil {
		if isUniqueViolation(err) {
			fail(c, http.StatusConflict, CodeUserExists, "username already exists")
			return
		}
		fail(c, http.StatusInternalServerError, CodeAssignRoles, "failed to create user")
		return
	}
	if err := seed.SyncUserRoles(a.Enforcer, seed.CasbinSub(user.Kind, user.ID), roles); err != nil {
		fail(c, http.StatusInternalServerError, CodeSyncRBAC, "failed to create user")
		return
	}
	user.Roles = roles
	raw, hash, err := mailer.NewResetToken()
	if err != nil {
		fail(c, http.StatusInternalServerError, CodeAssignRoles, "failed to create user")
		return
	}
	tok := models.PasswordResetToken{
		UserID: user.ID, UserKind: kind, Purpose: models.TokenPurposeVerify,
		TokenHash: hash, ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := a.DB.Create(&tok).Error; err != nil {
		fail(c, http.StatusInternalServerError, CodeAssignRoles, "failed to create user")
		return
	}
	cfg, _ := mailer.Load(a.DB, a.Cfg.JWTSecret)
	if link, okLink := a.mailPublicLink(cfg.ResetBaseURL, c.GetHeader("Origin"), "http://127.0.0.1:5174", "/verify-email?token="+raw); okLink {
		subject, body := i18n.VerifyEmailMail(i18n.FromRequest(c.Request), link)
		_, _ = a.enqueueMail(mailer.EnqueueInput{
			Class: models.MailClassTransactional, User: &user, ToEmail: email,
			Subject: subject,
			Body:    body,
		})
	}
	out := gin.H{"pending": true, "email": email}
	if a.Cfg.DevMode {
		out["verifyToken"] = raw
	}
	ok(c, out)
}

type verifyEmailRequest struct {
	Token string `json:"token"`
}

func (a *App) handleVerifyEmail(c *gin.Context) {
	var req verifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, CodeInvalidBody, "invalid request body")
		return
	}
	if a.VerifyGuard != nil && !a.VerifyGuard.AllowIP(c.ClientIP()) {
		fail(c, http.StatusTooManyRequests, CodeVerifyRateLimited, "too many verify requests from this ip")
		return
	}
	token := strings.TrimSpace(req.Token)
	if token != "" && a.VerifyTokenGuard != nil && !a.VerifyTokenGuard.AllowIP(mailer.HashToken(token)) {
		fail(c, http.StatusTooManyRequests, CodeVerifyRateLimited, "too many verify requests from this ip")
		return
	}
	if token == "" {
		fail(c, http.StatusBadRequest, CodeResetTokenInvalid, "invalid or expired reset token")
		return
	}
	var row models.PasswordResetToken
	err := a.DB.Where("token_hash = ?", mailer.HashToken(token)).First(&row).Error
	if err != nil || row.UsedAt != nil || time.Now().After(row.ExpiresAt) {
		fail(c, http.StatusBadRequest, CodeResetTokenInvalid, "invalid or expired reset token")
		return
	}
	kind := models.NormalizeUserKind(row.UserKind)
	if a.rejectWebKindMaintenance(c, kind) {
		return
	}
	var user models.User
	if err := a.loadAccount(kind, &user, row.UserID); err != nil {
		fail(c, http.StatusBadRequest, CodeResetTokenInvalid, "invalid or expired reset token")
		return
	}
	now := time.Now()
	if row.Purpose == models.TokenPurposeEmailChange {
		email := normalizeEmail(user.PendingEmail)
		if email == "" {
			fail(c, http.StatusBadRequest, CodeResetTokenInvalid, "invalid or expired reset token")
			return
		}
		if a.emailTaken(kind, email, user.ID) {
			fail(c, http.StatusConflict, CodeEmailExists, "email already exists")
			return
		}
		if err := a.updateAccount(&user, map[string]any{
			"email":          email,
			"pending_email":  "",
			"email_verified": true,
		}); err != nil {
			fail(c, http.StatusInternalServerError, CodeUpdateUser, "failed to update user")
			return
		}
		_ = a.DB.Model(&row).Update("used_at", now).Error
		user.Email = email
		user.PendingEmail = ""
		user.EmailVerified = true
		_ = models.AttachRoles(a.DB, kind, &user)
		ok(c, gin.H{"changed": true, "user": a.toUserDTO(user)})
		return
	}
	if row.Purpose != "" && row.Purpose != models.TokenPurposeVerify {
		fail(c, http.StatusBadRequest, CodeResetTokenInvalid, "invalid or expired reset token")
		return
	}
	if err := a.updateAccount(&user, map[string]any{"email_verified": true}); err != nil {
		fail(c, http.StatusInternalServerError, CodeUpdateUser, "failed to update user")
		return
	}
	_ = a.DB.Model(&row).Update("used_at", now).Error
	user.EmailVerified = true
	_ = models.AttachRoles(a.DB, kind, &user)
	a.finishLogin(c, user, c.ClientIP(), "verify-email")
}
