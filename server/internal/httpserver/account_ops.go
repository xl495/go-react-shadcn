package httpserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/i18n"
	"go-react-shadcn/internal/mailer"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/passwd"
	"go-react-shadcn/internal/secretbox"
	"go-react-shadcn/internal/seed"
	"go-react-shadcn/internal/totp"
	"gorm.io/gorm"
)

type copyRoleRequest struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

type batchStatusRequest struct {
	IDs    []uint `json:"ids"`
	Status string `json:"status"`
	Kind   string `json:"kind"`
}

type googleBindRequest struct {
	IDToken string `json:"idToken"`
}

type googleUnbindRequest struct {
	Password string `json:"password"`
	TotpCode string `json:"totpCode"`
}

type ownSessionDTO struct {
	ID        uint       `json:"id"`
	UserID    uint       `json:"userId"`
	UserKind  string     `json:"userKind"`
	IP        string     `json:"ip"`
	UserAgent string     `json:"userAgent"`
	ExpiresAt time.Time  `json:"expiresAt"`
	RevokedAt *time.Time `json:"revokedAt"`
	CreatedAt time.Time  `json:"createdAt"`
	Current   bool       `json:"current"`
}

type onlineSessionDTO struct {
	ID        uint       `json:"id"`
	UserID    uint       `json:"userId"`
	UserKind  string     `json:"userKind"`
	Username  string     `json:"username"`
	IP        string     `json:"ip"`
	UserAgent string     `json:"userAgent"`
	ExpiresAt time.Time  `json:"expiresAt"`
	RevokedAt *time.Time `json:"revokedAt"`
	CreatedAt time.Time  `json:"createdAt"`
}

func (a *App) rejectWebMaintenance(c *gin.Context, client string) bool {
	return a.rejectWebKindMaintenance(c, loginClientKind(client))
}

func (a *App) rejectWebKindMaintenance(c *gin.Context, kind string) bool {
	if !a.sysOn("app.maintenance", false) {
		return false
	}
	if models.NormalizeUserKind(kind) != models.UserKindWeb {
		return false
	}
	fail(c, http.StatusServiceUnavailable, CodeMaintenance, "the site is under maintenance")
	return true
}

func mustChangeAllowed(c *gin.Context) bool {
	path := c.FullPath()
	method := c.Request.Method
	if path == "/api/v1/auth/me" && method == http.MethodGet {
		return true
	}
	if path == "/api/v1/auth/password" && method == http.MethodPut {
		return true
	}
	if path == "/api/v1/auth/logout" && method == http.MethodPost {
		return true
	}
	if (path == "/api/v1/auth/menus" || path == "/api/v1/auth/web-menus") && method == http.MethodGet {
		return true
	}
	if strings.HasPrefix(path, "/api/v1/dicts/by/") && method == http.MethodGet {
		return true
	}
	return false
}

func (a *App) handleListOwnSessions(c *gin.Context) {
	user, _, err := a.currentAccount(c)
	if err != nil {
		fail(c, http.StatusNotFound, CodeUserMissingMe, "user not found")
		return
	}
	var rows []models.AuthSession
	if err := a.DB.Where("user_kind = ? AND user_id = ?", user.Kind, user.ID).Order("id desc").Limit(50).Find(&rows).Error; err != nil {
		fail(c, http.StatusInternalServerError, CodeListSessions, "failed to list sessions")
		return
	}
	claims := currentUser(c)
	currentJTI := ""
	if claims != nil {
		currentJTI = claims.ID
	}
	out := make([]ownSessionDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, ownSessionDTO{
			ID:        row.ID,
			UserID:    row.UserID,
			UserKind:  row.UserKind,
			IP:        row.IP,
			UserAgent: row.UserAgent,
			ExpiresAt: row.ExpiresAt,
			RevokedAt: row.RevokedAt,
			CreatedAt: row.CreatedAt,
			Current:   currentJTI != "" && row.JTI == currentJTI,
		})
	}
	ok(c, out)
}

func (a *App) handleRevokeOwnSession(c *gin.Context) {
	user, _, err := a.currentAccount(c)
	if err != nil {
		fail(c, http.StatusNotFound, CodeUserMissingMe, "user not found")
		return
	}
	var row models.AuthSession
	if err := a.DB.Where("id = ? AND user_kind = ? AND user_id = ?", c.Param("id"), user.Kind, user.ID).First(&row).Error; err != nil {
		fail(c, http.StatusNotFound, CodeSessionNotFound, "session not found")
		return
	}
	a.revokeAuthSessionJTI(row.JTI)
	ok(c, gin.H{"revoked": row.ID})
}

func (a *App) handleOwnLoginLogs(c *gin.Context) {
	claims := currentUser(c)
	if claims == nil {
		fail(c, http.StatusUnauthorized, CodeMissingToken, "missing bearer token")
		return
	}
	a.flushAuditLogs()
	p := parsePage(c, 20, 100)
	q := a.DB.Model(&models.LoginLog{}).Where("username = ? AND user_kind = ?", claims.Username, claimsKind(claims))
	total, okCount := countOrFail(c, q, CodeListLoginLogs, "failed to list login logs")
	if !okCount {
		return
	}
	var rows []models.LoginLog
	if err := q.Order("id desc").Offset(p.Offset()).Limit(p.PageSize).Find(&rows).Error; err != nil {
		fail(c, http.StatusInternalServerError, CodeListLoginLogs, "failed to list login logs")
		return
	}
	ok(c, pageResult[models.LoginLog]{Items: rows, Total: total, Page: p.Page, PageSize: p.PageSize})
}

func (a *App) handleResetUserPassword(c *gin.Context) {
	user, found := a.loadUserInScope(c, c.Param("id"))
	if !found {
		return
	}
	plain, err := randomTempPassword()
	if err != nil {
		fail(c, http.StatusInternalServerError, CodeHashPassword, "failed to hash password")
		return
	}
	hash, err := passwd.Hash(plain)
	if err != nil {
		fail(c, http.StatusInternalServerError, CodeHashPassword, "failed to hash password")
		return
	}
	if err := a.updateAccount(&user, map[string]any{
		"password_hash":        hash,
		"must_change_password": true,
		"token_version":        gorm.Expr("token_version + 1"),
		"failed_login_count":   0,
		"locked_until":         nil,
	}); err != nil {
		fail(c, http.StatusInternalServerError, CodeUpdateUser, "failed to update user")
		return
	}
	a.revokeAuthSessions(user.Kind, user.ID)
	a.sessions.invalidate(user.Kind, user.ID)
	a.notify(user.Kind, user.ID, "password", "密码已重置", "下次登录需修改密码", "user", user.ID)
	ok(c, gin.H{"temporaryPassword": plain, "mustChangePassword": true})
}

func (a *App) handleUnlockUser(c *gin.Context) {
	user, found := a.loadUserInScope(c, c.Param("id"))
	if !found {
		return
	}
	if err := a.updateAccount(&user, map[string]any{
		"locked_until":       nil,
		"failed_login_count": 0,
	}); err != nil {
		fail(c, http.StatusInternalServerError, CodeUpdateUser, "failed to update user")
		return
	}
	a.sessions.invalidate(user.Kind, user.ID)
	ok(c, gin.H{"unlocked": user.ID})
}

func (a *App) handleCopyRole(c *gin.Context) {
	var src models.Role
	if err := a.DB.Preload("Permissions").First(&src, c.Param("id")).Error; err != nil {
		fail(c, http.StatusNotFound, CodeRoleNotFound, "role not found")
		return
	}
	var req copyRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Code) == "" {
		fail(c, http.StatusBadRequest, CodeRoleNameRequired, "name and code required")
		return
	}
	code := strings.TrimSpace(req.Code)
	if isBuiltinRoleCode(code) {
		fail(c, http.StatusBadRequest, CodeRoleNameRequired, "name and code required")
		return
	}
	role := models.Role{
		Name:        strings.TrimSpace(req.Name),
		Code:        code,
		Description: src.Description,
		DataScope:   src.DataScope,
	}
	perms := src.Permissions
	if err := a.withTx(func(tx *gorm.DB) error {
		if err := tx.Create(&role).Error; err != nil {
			return err
		}
		return tx.Model(&role).Association("Permissions").Replace(perms)
	}); err != nil {
		if isUniqueViolation(err) {
			fail(c, http.StatusConflict, CodeRoleExists, "role code already exists")
			return
		}
		fail(c, http.StatusInternalServerError, 50021, "failed to create role")
		return
	}
	if err := seed.SyncRolePolicies(a.Enforcer, role.Code, perms); err != nil {
		fail(c, http.StatusInternalServerError, CodeSyncRBAC, "failed to sync rbac")
		return
	}
	role.Permissions = perms
	ok(c, toRoleDTO(role, true))
}

func isBuiltinRoleCode(code string) bool {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case seed.RoleAdmin, seed.RoleViewer, seed.RoleOperator, seed.RoleMember:
		return true
	default:
		return false
	}
}

func (a *App) handleBatchUserStatus(c *gin.Context) {
	var req batchStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, CodeInvalidUserBody, "invalid request body")
		return
	}
	status := strings.TrimSpace(req.Status)
	if status != "active" && status != "disabled" {
		fail(c, http.StatusBadRequest, CodeInvalidBatchStatus, "invalid batch status")
		return
	}
	kind := models.UserKindAdmin
	if strings.TrimSpace(req.Kind) != "" {
		kind = models.NormalizeUserKind(req.Kind)
	}
	actor, okActor := a.loadActor(c)
	if !okActor {
		return
	}
	updated := make([]uint, 0, len(req.IDs))
	skipped := make([]uint, 0)
	for _, id := range parseIDs(req.IDs) {
		q := a.applyUserDataScope(a.accounts(kind), actor, kind)
		var user models.User
		if err := q.First(&user, id).Error; err != nil {
			skipped = append(skipped, id)
			continue
		}
		if err := models.AttachRoles(a.DB, kind, &user); err != nil {
			skipped = append(skipped, id)
			continue
		}
		if status == "disabled" && kind == models.UserKindAdmin && hasRole(user.Roles, seed.RoleAdmin) {
			n, err := a.countActiveAdmins(user.ID)
			if err != nil {
				fail(c, http.StatusInternalServerError, CodeUpdateUser, "failed to update user")
				return
			}
			if n == 0 {
				fail(c, http.StatusBadRequest, CodeCannotDisableLastAdmin, "cannot disable the last active admin")
				return
			}
		}
		values := map[string]any{"status": status}
		if status == "disabled" {
			values["token_version"] = gorm.Expr("token_version + 1")
		}
		if err := a.updateAccount(&user, values); err != nil {
			fail(c, http.StatusInternalServerError, CodeUpdateUser, "failed to update user")
			return
		}
		if status == "disabled" {
			a.revokeAuthSessions(kind, user.ID)
			a.sessions.invalidate(kind, user.ID)
			a.notify(kind, user.ID, "status", "账号已停用", "", "user", user.ID)
		} else {
			a.sessions.invalidate(kind, user.ID)
			a.notify(kind, user.ID, "status", "账号已启用", "", "user", user.ID)
		}
		updated = append(updated, user.ID)
	}
	ok(c, gin.H{"updated": updated, "skipped": skipped})
}

func (a *App) countActiveAdmins(exceptID uint) (int64, error) {
	q := a.DB.Table(models.AdminUserTable).
		Joins("JOIN user_roles ON user_roles.user_id = "+models.AdminUserTable+".id").
		Joins("JOIN roles ON roles.id = user_roles.role_id AND roles.code = ?", seed.RoleAdmin).
		Where(models.AdminUserTable+".status = ?", "active")
	if exceptID > 0 {
		q = q.Where(models.AdminUserTable+".id <> ?", exceptID)
	}
	var n int64
	err := q.Distinct(models.AdminUserTable + ".id").Count(&n).Error
	return n, err
}

func (a *App) handleOnlineSessions(c *gin.Context) {
	now := time.Now()
	var rows []models.AuthSession
	if err := a.DB.Where("revoked_at IS NULL AND expires_at > ?", now).Order("id desc").Limit(200).Find(&rows).Error; err != nil {
		fail(c, http.StatusInternalServerError, CodeListSessions, "failed to list sessions")
		return
	}
	adminIDs := map[uint]struct{}{}
	webIDs := map[uint]struct{}{}
	for _, row := range rows {
		if row.UserKind == models.UserKindWeb {
			webIDs[row.UserID] = struct{}{}
		} else {
			adminIDs[row.UserID] = struct{}{}
		}
	}
	names := map[string]string{}
	loadNames := func(kind string, ids map[uint]struct{}) {
		if len(ids) == 0 {
			return
		}
		list := make([]uint, 0, len(ids))
		for id := range ids {
			list = append(list, id)
		}
		var users []models.User
		_ = a.accounts(kind).Select("id", "username").Where("id IN ?", list).Find(&users).Error
		for _, u := range users {
			names[kind+":"+formatUint(u.ID)] = u.Username
		}
	}
	loadNames(models.UserKindAdmin, adminIDs)
	loadNames(models.UserKindWeb, webIDs)
	out := make([]onlineSessionDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, onlineSessionDTO{
			ID:        row.ID,
			UserID:    row.UserID,
			UserKind:  row.UserKind,
			Username:  names[row.UserKind+":"+formatUint(row.UserID)],
			IP:        row.IP,
			UserAgent: row.UserAgent,
			ExpiresAt: row.ExpiresAt,
			RevokedAt: row.RevokedAt,
			CreatedAt: row.CreatedAt,
		})
	}
	ok(c, out)
}

func (a *App) handleGoogleBind(c *gin.Context) {
	if !a.googleEnabled() {
		fail(c, http.StatusServiceUnavailable, CodeGoogleDisabled, "google sign-in is not enabled")
		return
	}
	user, _, err := a.currentAccount(c)
	if err != nil {
		fail(c, http.StatusNotFound, CodeUserMissingMe, "user not found")
		return
	}
	var req googleBindRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.IDToken) == "" {
		fail(c, http.StatusBadRequest, CodeInvalidBody, "invalid request body")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()
	ident, err := a.googleVerifier().Verify(ctx, req.IDToken, a.googleClientID())
	if err != nil {
		fail(c, http.StatusUnauthorized, CodeGoogleTokenInvalid, "google sign-in failed")
		return
	}
	if ident.Email == "" || !ident.EmailVerified {
		fail(c, http.StatusUnauthorized, CodeGoogleEmailUnverified, "google email is not verified")
		return
	}
	if user.Email != "" && !strings.EqualFold(user.Email, ident.Email) {
		fail(c, http.StatusBadRequest, CodeGoogleEmailMismatch, "google account email does not match")
		return
	}
	if ident.Subject != "" {
		var other models.User
		err := a.accounts(user.Kind).Where("google_id = ? AND google_id <> '' AND id <> ?", ident.Subject, user.ID).First(&other).Error
		if err == nil {
			fail(c, http.StatusConflict, CodeGoogleAccountConflict, "this google account is already linked to another user")
			return
		}
	}
	updates := map[string]any{"google_id": ident.Subject}
	if user.Email == "" {
		updates["email"] = ident.Email
		updates["email_verified"] = true
	}
	if err := a.updateAccount(&user, updates); err != nil {
		fail(c, http.StatusInternalServerError, CodeUpdateProfile, "failed to update profile")
		return
	}
	_ = a.loadAccount(user.Kind, &user, user.ID)
	_ = models.AttachRoles(a.DB, user.Kind, &user)
	ok(c, a.toUserDTO(user))
}

func (a *App) handleGoogleUnbind(c *gin.Context) {
	user, _, err := a.currentAccount(c)
	if err != nil {
		fail(c, http.StatusNotFound, CodeUserMissingMe, "user not found")
		return
	}
	if user.GoogleID == "" {
		ok(c, a.toUserDTO(user))
		return
	}
	var req googleUnbindRequest
	_ = c.ShouldBindJSON(&req)
	if !a.confirmSensitiveAction(c, user, req.Password, req.TotpCode) {
		return
	}
	if err := a.updateAccount(&user, map[string]any{"google_id": ""}); err != nil {
		fail(c, http.StatusInternalServerError, CodeUpdateProfile, "failed to update profile")
		return
	}
	user.GoogleID = ""
	ok(c, a.toUserDTO(user))
}

func (a *App) beginEmailChange(c *gin.Context, user *models.User, newEmail string) (token string, okBegin bool) {
	if a.emailTaken(user.Kind, newEmail, user.ID) {
		fail(c, http.StatusConflict, CodeEmailExists, "email already exists")
		return "", false
	}
	raw, hash, err := mailer.NewResetToken()
	if err != nil {
		fail(c, http.StatusInternalServerError, CodeUpdateProfile, "failed to update profile")
		return "", false
	}
	row := models.PasswordResetToken{
		UserID:    user.ID,
		UserKind:  user.Kind,
		Purpose:   models.TokenPurposeEmailChange,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := a.withTx(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ? AND user_kind = ? AND purpose = ? AND used_at IS NULL", user.ID, user.Kind, models.TokenPurposeEmailChange).
			Delete(&models.PasswordResetToken{}).Error; err != nil {
			return err
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return models.Accounts(tx, user.Kind).Where("id = ?", user.ID).Update("pending_email", newEmail).Error
	}); err != nil {
		fail(c, http.StatusInternalServerError, CodeUpdateProfile, "failed to update profile")
		return "", false
	}
	user.PendingEmail = newEmail
	cfg, _ := mailer.Load(a.DB, a.Cfg.JWTSecret)
	if link, okLink := a.mailPublicLink(cfg.ResetBaseURL, c.GetHeader("Origin"), "http://127.0.0.1:5174", "/verify-email?token="+raw); okLink {
		subject, body := i18n.VerifyEmailMail(i18n.FromRequest(c.Request), link)
		_, _ = a.enqueueMail(mailer.EnqueueInput{
			Class: models.MailClassTransactional, User: user, ToEmail: newEmail,
			Subject: subject,
			Body:    body,
		})
	}
	if a.Cfg.DevMode {
		return raw, true
	}
	return "", true
}

func (a *App) confirmSensitiveAction(c *gin.Context, user models.User, password, totpCode string) bool {
	if user.TotpEnabled {
		secret := secretbox.MustOpen(a.Cfg.JWTSecret, user.TotpSecret)
		if secret == "" || !totp.Valid(secret, totpCode, time.Now()) {
			fail(c, http.StatusBadRequest, CodeInvalidTotp, "invalid totp code")
			return false
		}
		return true
	}
	if strings.TrimSpace(password) == "" {
		fail(c, http.StatusBadRequest, CodeGoogleNeedPassword, "set a password before unbinding google")
		return false
	}
	if !passwd.Match(user.PasswordHash, password) {
		fail(c, http.StatusBadRequest, CodeWrongPassword, "current password is wrong")
		return false
	}
	return true
}

func (a *App) rejectLastAdminLoss(c *gin.Context, user models.User, nextStatus string, nextRoles []models.Role, deleting bool) bool {
	if models.NormalizeUserKind(user.Kind) != models.UserKindAdmin || !hasRole(user.Roles, seed.RoleAdmin) {
		return false
	}
	losing := deleting
	if nextStatus != "" && nextStatus != "active" {
		losing = true
	}
	if nextRoles != nil && !hasRole(nextRoles, seed.RoleAdmin) {
		losing = true
	}
	if !losing {
		return false
	}
	n, err := a.countActiveAdmins(user.ID)
	if err != nil {
		fail(c, http.StatusInternalServerError, CodeUpdateUser, "failed to update user")
		return true
	}
	if n == 0 {
		fail(c, http.StatusBadRequest, CodeCannotDisableLastAdmin, "cannot disable the last active admin")
		return true
	}
	return false
}

func randomTempPassword() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "Aa1" + hex.EncodeToString(buf), nil
}
