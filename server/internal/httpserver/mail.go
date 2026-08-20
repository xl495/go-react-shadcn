package httpserver

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/mailer"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/passwd"
	"gorm.io/gorm"
)

type forgotPasswordRequest struct {
	Email string `json:"email"`
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

	var user models.User
	err := a.DB.Where("lower(email) = ? AND email <> '' AND status = ?", email, "active").First(&user).Error
	if err != nil {
		return
	}
	raw, hash, err := mailer.NewResetToken()
	if err != nil {
		slog.Error("forgot password token", "error", err)
		return
	}
	_ = a.DB.Where("user_id = ? AND used_at IS NULL", user.ID).Delete(&models.PasswordResetToken{}).Error
	row := models.PasswordResetToken{
		UserID:    user.ID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(mailer.ResetTTL),
	}
	if err := a.DB.Create(&row).Error; err != nil {
		slog.Error("forgot password persist", "error", err)
		return
	}
	cfg, _ := mailer.Load(a.DB)
	name := user.Nickname
	if name == "" {
		name = user.Username
	}
	link := mailer.ResetLink(cfg.ResetBaseURL, c.GetHeader("Origin"), raw)
	body := fmt.Sprintf("您好 %s，\n\n请在 30 分钟内点击下面的链接重置密码：\n%s\n\n若非本人操作，请忽略此邮件。\n", name, link)
	if _, err := a.enqueueMail(mailer.EnqueueInput{
		Class:   models.MailClassTransactional,
		User:    &user,
		ToEmail: user.Email,
		Subject: "重置 Latch 密码",
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
	token := strings.TrimSpace(req.Token)
	if token == "" {
		fail(c, http.StatusBadRequest, CodeResetTokenInvalid, "invalid or expired reset token")
		return
	}
	if len(req.NewPassword) < 8 {
		fail(c, http.StatusBadRequest, CodeNewPasswordShort, "password must be at least 8 characters")
		return
	}
	var row models.PasswordResetToken
	err := a.DB.Where("token_hash = ?", mailer.HashToken(token)).First(&row).Error
	if err != nil || row.UsedAt != nil || time.Now().After(row.ExpiresAt) {
		fail(c, http.StatusBadRequest, CodeResetTokenInvalid, "invalid or expired reset token")
		return
	}
	var user models.User
	if err := a.DB.First(&user, row.UserID).Error; err != nil {
		fail(c, http.StatusBadRequest, CodeResetTokenInvalid, "invalid or expired reset token")
		return
	}
	if user.Status != "active" {
		fail(c, http.StatusBadRequest, CodeResetTokenInvalid, "invalid or expired reset token")
		return
	}
	hash, err := passwd.Hash(req.NewPassword)
	if err != nil {
		fail(c, http.StatusInternalServerError, CodeHashPassword, "failed to hash password")
		return
	}
	now := time.Now()
	err = a.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&user).Updates(map[string]any{
			"password_hash": hash,
			"token_version": user.TokenVersion + 1,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&row).Update("used_at", now).Error; err != nil {
			return err
		}
		return tx.Where("user_id = ? AND id <> ? AND used_at IS NULL", user.ID, row.ID).Delete(&models.PasswordResetToken{}).Error
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, CodeChangePassword, "failed to change password")
		return
	}
	a.sessions.invalidate(user.ID)
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
	cfg, _ := mailer.Load(a.DB)
	job, err := a.enqueueMail(mailer.EnqueueInput{
		Class:   models.MailClassTransactional,
		ToEmail: to,
		Subject: "Latch 测试邮件",
		Body:    "这是来自 Latch 的测试邮件，说明当前 SMTP 配置可用。",
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
