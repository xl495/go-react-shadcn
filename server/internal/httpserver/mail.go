package httpserver

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/i18n"
	"go-react-shadcn/internal/mailer"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/passwd"
	"gorm.io/gorm"
)

type forgotPasswordRequest struct {
	Email  string `json:"email"`
	Client string `json:"client"`
	challengeInput
}

type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"newPassword"`
}

type testMailRequest struct {
	To string `json:"to"`
}

func (a *App) handleForgotPassword(c *gin.Context) {
	var req forgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, CodeInvalidBody, "invalid request body")
		return
	}
	if a.rejectWebMaintenance(c, req.Client) {
		return
	}
	if a.ForgotGuard != nil && !a.ForgotGuard.AllowIP(c.ClientIP()) {
		fail(c, http.StatusTooManyRequests, CodeForgotRateLimited, "too many reset requests from this ip")
		return
	}
	if !a.requireCaptcha(c, req.challengeInput, "forgot") {
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || !mailer.ValidAddress(email) {
		fail(c, http.StatusBadRequest, CodeEmailRequired, "email required")
		return
	}

	ok(c, gin.H{"sent": true})

	kind := loginClientKind(req.Client)
	var user models.User
	err := a.accounts(kind).Where("lower(email) = ? AND email <> '' AND status = ?", email, "active").First(&user).Error
	if err != nil {
		return
	}
	user.Kind = kind
	raw, hash, err := mailer.NewResetToken()
	if err != nil {
		slog.Error("forgot password token", "error", err)
		return
	}
	_ = a.DB.Where("user_id = ? AND user_kind = ? AND used_at IS NULL AND (purpose = ? OR purpose = '')", user.ID, kind, models.TokenPurposeReset).
		Delete(&models.PasswordResetToken{}).Error
	row := models.PasswordResetToken{
		UserID:    user.ID,
		UserKind:  kind,
		Purpose:   models.TokenPurposeReset,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(mailer.ResetTTL),
	}
	if err := a.DB.Create(&row).Error; err != nil {
		slog.Error("forgot password persist", "error", err)
		return
	}
	cfg, _ := mailer.Load(a.DB, a.Cfg.JWTSecret)
	name := user.Nickname
	if name == "" {
		name = user.Username
	}
	link, okLink := a.mailPublicLink(cfg.ResetBaseURL, c.GetHeader("Origin"), mailFallbackOrigin(kind), "/reset-password?token="+raw)
	if !okLink {
		slog.Error("forgot password missing mail.reset_base_url in production")
		return
	}
	subject, body := i18n.ResetPasswordMail(i18n.FromRequest(c.Request), name, link)
	if _, err := a.enqueueMail(mailer.EnqueueInput{
		Class:   models.MailClassTransactional,
		User:    &user,
		ToEmail: user.Email,
		Subject: subject,
		Body:    body,
	}); err != nil {
		slog.Error("forgot password enqueue", "error", err, "user", user.Username)
		return
	}
	a.flushMail(time.Now(), 8)
}

func (a *App) handleResetPassword(c *gin.Context) {
	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, CodeInvalidBody, "invalid request body")
		return
	}
	if a.ResetGuard != nil && !a.ResetGuard.AllowIP(c.ClientIP()) {
		fail(c, http.StatusTooManyRequests, CodeForgotRateLimited, "too many reset requests from this ip")
		return
	}
	token := strings.TrimSpace(req.Token)
	if token != "" && a.ResetTokenGuard != nil && !a.ResetTokenGuard.AllowIP(mailer.HashToken(token)) {
		fail(c, http.StatusTooManyRequests, CodeForgotRateLimited, "too many reset requests from this ip")
		return
	}
	if token == "" {
		fail(c, http.StatusBadRequest, CodeResetTokenInvalid, "invalid or expired reset token")
		return
	}
	var row models.PasswordResetToken
	err := a.DB.Where("token_hash = ?", mailer.HashToken(token)).First(&row).Error
	if err != nil || row.UsedAt != nil || time.Now().After(row.ExpiresAt) || (row.Purpose != "" && row.Purpose != models.TokenPurposeReset) {
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
	if user.Status != "active" {
		fail(c, http.StatusBadRequest, CodeResetTokenInvalid, "invalid or expired reset token")
		return
	}
	if a.failIfWeakPassword(c, req.NewPassword, user.Username) {
		return
	}
	hash, err := passwd.Hash(req.NewPassword)
	if err != nil {
		fail(c, http.StatusInternalServerError, CodeHashPassword, "failed to hash password")
		return
	}
	now := time.Now()
	err = a.DB.Transaction(func(tx *gorm.DB) error {
		if err := models.Accounts(tx, kind).Where("id = ?", user.ID).Updates(map[string]any{
			"password_hash":        hash,
			"token_version":        user.TokenVersion + 1,
			"locked_until":         nil,
			"failed_login_count":   0,
			"must_change_password": false,
			"must_set_password":    false,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&row).Update("used_at", now).Error; err != nil {
			return err
		}
		return tx.Where("user_id = ? AND user_kind = ? AND id <> ? AND used_at IS NULL AND (purpose = ? OR purpose = '')", user.ID, kind, row.ID, models.TokenPurposeReset).
			Delete(&models.PasswordResetToken{}).Error
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, CodeChangePassword, "failed to change password")
		return
	}
	a.revokeAuthSessions(kind, user.ID)
	a.sessions.invalidate(kind, user.ID)
	ok(c, gin.H{"reset": true})
}

func (a *App) handleTestMail(c *gin.Context) {
	var req testMailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, CodeInvalidBody, "invalid request body")
		return
	}
	to := strings.TrimSpace(req.To)
	if to == "" {
		fail(c, http.StatusBadRequest, CodeMailRecipient, "recipient required")
		return
	}
	if !mailer.ValidAddress(to) {
		fail(c, http.StatusBadRequest, CodeMailRecipient, "invalid email address")
		return
	}
	cfg, _ := mailer.Load(a.DB, a.Cfg.JWTSecret)
	job, err := a.enqueueMail(mailer.EnqueueInput{
		Class:   models.MailClassTransactional,
		ToEmail: to,
		Subject: "gra 测试邮件",
		Body:    "这是来自 gra 的测试邮件，说明当前 SMTP 配置可用。",
	})
	if err != nil {
		a.failMail(c, err)
		return
	}
	a.flushMail(time.Now(), 8)
	if err := a.DB.First(job, job.ID).Error; err == nil {
		if job.Status == models.MailStatusSent {
			ok(c, gin.H{"sent": true, "to": to})
			return
		}
		if job.LastError != "" {
			a.failMail(c, mailErrFromMessage(job.LastError))
			return
		}
	}
	if !cfg.Enabled {
		fail(c, http.StatusServiceUnavailable, CodeMailDisabled, "mail is not enabled")
		return
	}
	ok(c, gin.H{"sent": true, "to": to, "queued": true})
}

func (a *App) enqueueMail(in mailer.EnqueueInput) (*models.MailJob, error) {
	if a.MailQ == nil {
		return nil, errors.New("mail queue unavailable")
	}
	if a.Mail != nil {
		a.MailQ.SetSender(a.Mail)
	}
	return a.MailQ.Enqueue(in)
}

func (a *App) flushMail(now time.Time, n int) {
	if a.MailQ == nil {
		return
	}
	if a.Mail != nil {
		a.MailQ.SetSender(a.Mail)
	}
	a.MailQ.ProcessAvailable(now, n)
}

func mailErrFromMessage(msg string) error {
	switch {
	case strings.Contains(msg, mailer.ErrDisabled.Error()):
		return mailer.ErrDisabled
	case strings.Contains(msg, mailer.ErrIncomplete.Error()):
		return mailer.ErrIncomplete
	case strings.Contains(msg, mailer.ErrInvalidAddr.Error()):
		return mailer.ErrInvalidAddr
	default:
		return errors.New(msg)
	}
}

func (a *App) failMail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, mailer.ErrDisabled):
		fail(c, http.StatusServiceUnavailable, CodeMailDisabled, "mail is not enabled")
	case errors.Is(err, mailer.ErrIncomplete):
		fail(c, http.StatusBadRequest, CodeMailIncomplete, "mail is not configured")
	case errors.Is(err, mailer.ErrInvalidAddr):
		fail(c, http.StatusBadRequest, CodeMailRecipient, "invalid email address")
	default:
		fail(c, http.StatusInternalServerError, CodeSendMail, "failed to send mail")
	}
}

func (a *App) allowedLinkOrigins() []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, 8)
	add := func(raw string) {
		item := mailer.NormalizeOrigin(raw)
		if item == "" {
			return
		}
		if _, ok := seen[item]; ok {
			return
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	for _, part := range strings.Split(a.Cfg.CORSOrigin, ",") {
		add(part)
	}
	if a.Cfg.DevMode {
		add("http://127.0.0.1:5173")
		add("http://localhost:5173")
		add("http://127.0.0.1:5174")
		add("http://localhost:5174")
	}
	return out
}

func (a *App) mailPublicLink(configured, origin, fallback, suffix string) (string, bool) {
	configuredBase := mailer.NormalizeOrigin(configured)
	if configuredBase == "" && !a.Cfg.DevMode {
		return "", false
	}
	base := ""
	if mailer.OriginAllowed(origin, a.allowedLinkOrigins()) {
		base = mailer.NormalizeOrigin(origin)
	} else {
		base = mailer.LinkBase(configured, origin, a.allowedLinkOrigins(), fallback)
	}
	if base == "" {
		return "", false
	}
	return base + suffix, true
}
