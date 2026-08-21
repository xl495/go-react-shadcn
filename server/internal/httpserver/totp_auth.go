package httpserver

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/secretbox"
	"go-react-shadcn/internal/totp"
)

type totpTicket struct {
	userID  uint
	kind    string
	purpose string
	secret  string
	expires time.Time
}

func (a *App) issueTotpTicket(user models.User, purpose, pendingSecret string) string {
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	id := hex.EncodeToString(buf)
	a.totpMu.Lock()
	defer a.totpMu.Unlock()
	if a.totpTickets == nil {
		a.totpTickets = map[string]totpTicket{}
	}
	a.totpTickets[id] = totpTicket{
		userID: user.ID, kind: models.NormalizeUserKind(user.Kind), purpose: purpose,
		secret: pendingSecret, expires: time.Now().Add(10 * time.Minute),
	}
	return id
}

func (a *App) takeTotpTicket(id, purpose string) (totpTicket, bool) {
	a.totpMu.Lock()
	defer a.totpMu.Unlock()
	row, ok := a.totpTickets[id]
	if !ok || time.Now().After(row.expires) || (purpose != "" && row.purpose != purpose) {
		return totpTicket{}, false
	}
	delete(a.totpTickets, id)
	return row, true
}

func (a *App) peekTotpTicket(id string) (totpTicket, bool) {
	a.totpMu.Lock()
	defer a.totpMu.Unlock()
	row, ok := a.totpTickets[id]
	if !ok || time.Now().After(row.expires) {
		return totpTicket{}, false
	}
	return row, true
}

func (a *App) putTotpTicket(id string, row totpTicket) {
	a.totpMu.Lock()
	defer a.totpMu.Unlock()
	if a.totpTickets == nil {
		a.totpTickets = map[string]totpTicket{}
	}
	a.totpTickets[id] = row
}

func (a *App) sweepTotpTickets(now time.Time) {
	a.totpMu.Lock()
	defer a.totpMu.Unlock()
	for id, row := range a.totpTickets {
		if now.After(row.expires) {
			delete(a.totpTickets, id)
		}
	}
}

func (a *App) totpNeeded(user models.User) (required, enroll bool) {
	if user.TotpEnabled {
		return true, false
	}
	if models.NormalizeUserKind(user.Kind) == models.UserKindAdmin && a.sysOn("auth.admin_totp_required", false) {
		return true, true
	}
	return false, false
}

func (a *App) handleTotpSetup(c *gin.Context) {
	var req totpCodeRequest
	_ = c.ShouldBindJSON(&req)
	user, ticketID, okUser := a.totpActor(c, strings.TrimSpace(req.Ticket))
	if !okUser {
		return
	}
	if user.TotpEnabled {
		fail(c, http.StatusBadRequest, CodeTotpAlreadyEnabled, "totp already enabled")
		return
	}
	secret, err := totp.RandomSecret()
	if err != nil {
		fail(c, http.StatusInternalServerError, CodeTokenIssue, "failed to issue token")
		return
	}
	if ticketID == "" {
		ticketID = a.issueTotpTicket(user, "enroll", secret)
	} else {
		row, ok := a.peekTotpTicket(ticketID)
		if !ok || row.userID != user.ID {
			fail(c, http.StatusUnauthorized, CodeTotpTicketInvalid, "invalid or expired totp ticket")
			return
		}
		row.secret = secret
		row.purpose = "enroll"
		a.putTotpTicket(ticketID, row)
	}
	ok(c, gin.H{
		"ticket":     ticketID,
		"secret":     secret,
		"otpauthUri": totp.URI(user.Username, secret),
	})
}

type totpCodeRequest struct {
	Ticket      string `json:"ticket"`
	Code        string `json:"code"`
	RecoveryCode string `json:"recoveryCode"`
}

func (a *App) handleTotpConfirm(c *gin.Context) {
	var req totpCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Code) == "" {
		fail(c, http.StatusBadRequest, CodeInvalidTotp, "invalid totp code")
		return
	}
	user, ticketID, okUser := a.totpActor(c, strings.TrimSpace(req.Ticket))
	if !okUser {
		return
	}
	if ticketID == "" {
		fail(c, http.StatusUnauthorized, CodeTotpTicketInvalid, "invalid or expired totp ticket")
		return
	}
	row, taken := a.takeTotpTicket(ticketID, "enroll")
	if !taken || row.userID != user.ID || row.secret == "" {
		fail(c, http.StatusUnauthorized, CodeTotpTicketInvalid, "invalid or expired totp ticket")
		return
	}
	if !totp.Valid(row.secret, req.Code, time.Now()) {
		a.putTotpTicket(ticketID, row)
		fail(c, http.StatusBadRequest, CodeInvalidTotp, "invalid totp code")
		return
	}
	sealed, err := secretbox.Seal(a.Cfg.JWTSecret, row.secret)
	if err != nil {
		fail(c, http.StatusInternalServerError, CodeTokenIssue, "failed to issue token")
		return
	}
	now := time.Now()
	if err := a.updateAccount(&user, map[string]any{
		"totp_enabled":     true,
		"totp_secret":      sealed,
		"totp_verified_at": now,
	}); err != nil {
		fail(c, http.StatusInternalServerError, CodeUpdateUser, "failed to update user")
		return
	}
	user.TotpEnabled = true
	user.TotpSecret = sealed
	codes := a.replaceRecoveryCodes(user)
	if currentUser(c) != nil {
		ok(c, gin.H{"enabled": true, "recoveryCodes": codes})
		return
	}
	a.completeLogin(c, user, c.ClientIP(), "totp-enroll", codes)
}

func (a *App) handleTotpVerify(c *gin.Context) {
	var req totpCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Ticket) == "" {
		fail(c, http.StatusBadRequest, CodeTotpTicketInvalid, "invalid or expired totp ticket")
		return
	}
	row, ok := a.peekTotpTicket(req.Ticket)
	if !ok {
		fail(c, http.StatusUnauthorized, CodeTotpTicketInvalid, "invalid or expired totp ticket")
		return
	}
	var user models.User
	if err := a.loadAccount(row.kind, &user, row.userID); err != nil {
		fail(c, http.StatusUnauthorized, CodeTotpTicketInvalid, "invalid or expired totp ticket")
		return
	}
	if strings.TrimSpace(req.RecoveryCode) != "" {
		if !a.consumeRecoveryCode(user, req.RecoveryCode) {
			fail(c, http.StatusBadRequest, CodeInvalidTotp, "invalid totp code")
			return
		}
		_, _ = a.takeTotpTicket(req.Ticket, "")
		a.completeLogin(c, user, c.ClientIP(), "totp-recovery", nil)
		return
	}
	secret := secretbox.MustOpen(a.Cfg.JWTSecret, user.TotpSecret)
	if secret == "" || !totp.Valid(secret, req.Code, time.Now()) {
		fail(c, http.StatusBadRequest, CodeInvalidTotp, "invalid totp code")
		return
	}
	_, _ = a.takeTotpTicket(req.Ticket, "")
	a.completeLogin(c, user, c.ClientIP(), "totp", nil)
}

func (a *App) handleTotpDisable(c *gin.Context) {
	user, _, err := a.currentAccount(c)
	if err != nil {
		fail(c, http.StatusUnauthorized, CodeMissingToken, "missing bearer token")
		return
	}
	var req totpCodeRequest
	_ = c.ShouldBindJSON(&req)
	secret := secretbox.MustOpen(a.Cfg.JWTSecret, user.TotpSecret)
	okCode := user.TotpEnabled && secret != "" && totp.Valid(secret, req.Code, time.Now())
	okRecovery := a.consumeRecoveryCode(user, req.RecoveryCode)
	if !okCode && !okRecovery {
		fail(c, http.StatusBadRequest, CodeInvalidTotp, "invalid totp code")
		return
	}
	_ = a.DB.Where("user_kind = ? AND user_id = ?", user.Kind, user.ID).Delete(&models.TotpRecoveryCode{}).Error
	if err := a.updateAccount(&user, map[string]any{
		"totp_enabled": false, "totp_secret": "", "totp_verified_at": nil,
	}); err != nil {
		fail(c, http.StatusInternalServerError, CodeUpdateUser, "failed to update user")
		return
	}
	ok(c, gin.H{"enabled": false})
}

func (a *App) handleTotpRecovery(c *gin.Context) {
	user, _, err := a.currentAccount(c)
	if err != nil {
		fail(c, http.StatusUnauthorized, CodeMissingToken, "missing bearer token")
		return
	}
	var req totpCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, CodeInvalidTotp, "invalid totp code")
		return
	}
	secret := secretbox.MustOpen(a.Cfg.JWTSecret, user.TotpSecret)
	if !user.TotpEnabled || secret == "" || !totp.Valid(secret, req.Code, time.Now()) {
		fail(c, http.StatusBadRequest, CodeInvalidTotp, "invalid totp code")
		return
	}
	ok(c, gin.H{"recoveryCodes": a.replaceRecoveryCodes(user)})
}

func (a *App) totpActor(c *gin.Context, ticket string) (models.User, string, bool) {
	if claims := currentUser(c); claims != nil {
		user, _, err := a.currentAccount(c)
		if err != nil {
			fail(c, http.StatusUnauthorized, CodeMissingToken, "missing bearer token")
			return models.User{}, "", false
		}
		return user, strings.TrimSpace(ticket), true
	}
	ticket = strings.TrimSpace(ticket)
	if ticket == "" {
		ticket = strings.TrimSpace(c.Query("ticket"))
	}
	row, ok := a.peekTotpTicket(ticket)
	if !ok {
		fail(c, http.StatusUnauthorized, CodeTotpTicketInvalid, "invalid or expired totp ticket")
		return models.User{}, "", false
	}
	var user models.User
	if err := a.loadAccount(row.kind, &user, row.userID); err != nil {
		fail(c, http.StatusUnauthorized, CodeTotpTicketInvalid, "invalid or expired totp ticket")
		return models.User{}, "", false
	}
	return user, ticket, true
}

func (a *App) replaceRecoveryCodes(user models.User) []string {
	_ = a.DB.Where("user_kind = ? AND user_id = ?", user.Kind, user.ID).Delete(&models.TotpRecoveryCode{}).Error
	plain := make([]string, 8)
	rows := make([]models.TotpRecoveryCode, 8)
	now := time.Now()
	for i := range plain {
		buf := make([]byte, 5)
		_, _ = rand.Read(buf)
		plain[i] = strings.ToLower(hex.EncodeToString(buf))
		rows[i] = models.TotpRecoveryCode{
			UserID: user.ID, UserKind: models.NormalizeUserKind(user.Kind),
			CodeHash: recoveryHash(user, plain[i]), CreatedAt: now,
		}
	}
	_ = a.DB.Create(&rows).Error
	return plain
}

func (a *App) consumeRecoveryCode(user models.User, code string) bool {
	code = strings.TrimSpace(strings.ToLower(code))
	if code == "" {
		return false
	}
	hash := recoveryHash(user, code)
	var row models.TotpRecoveryCode
	if err := a.DB.Where("user_kind = ? AND user_id = ? AND code_hash = ? AND used_at IS NULL", user.Kind, user.ID, hash).First(&row).Error; err != nil {
		return false
	}
	now := time.Now()
	return a.DB.Model(&row).Update("used_at", now).Error == nil
}

func recoveryHash(user models.User, code string) string {
	sum := sha256.Sum256([]byte(user.Kind + "|" + hex.EncodeToString([]byte{byte(user.ID)}) + "|" + code))
	return hex.EncodeToString(sum[:])
}
