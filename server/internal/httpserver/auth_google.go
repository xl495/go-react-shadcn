package httpserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/googleid"
	"go-react-shadcn/internal/mailer"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/passwd"
	"go-react-shadcn/internal/security"
	"go-react-shadcn/internal/seed"
	"gorm.io/gorm"
)

type publicAuthSettings struct {
	GoogleEnabled         bool   `json:"googleEnabled"`
	GoogleRegisterEnabled bool   `json:"googleRegisterEnabled"`
	RegisterEnabled       bool   `json:"registerEnabled"`
	GoogleClientID        string `json:"googleClientId"`
	CaptchaProvider       string `json:"captchaProvider"`
	RecaptchaSiteKeyV3    string `json:"recaptchaSiteKeyV3"`
	RecaptchaSiteKeyV2    string `json:"recaptchaSiteKeyV2"`
	TurnstileSiteKey      string `json:"turnstileSiteKey"`
	Maintenance           bool   `json:"maintenance"`
}

type googleAuthRequest struct {
	IDToken string `json:"idToken"`
	Client  string `json:"client"`
}

func (a *App) handleAuthSettings(c *gin.Context) {
	provider := a.captchaProvider()
	clientID := a.googleClientID()
	enabled := a.googleEnabled()
	ok(c, publicAuthSettings{
		GoogleEnabled:         enabled,
		GoogleRegisterEnabled: enabled && a.sysOn("auth.google_register_enabled", false),
		RegisterEnabled:       a.sysOn("auth.register_enabled", true),
		GoogleClientID:        clientID,
		CaptchaProvider:       provider,
		RecaptchaSiteKeyV3:    a.sysValue("auth.recaptcha_site_key_v3"),
		RecaptchaSiteKeyV2:    a.sysValue("auth.recaptcha_site_key_v2"),
		TurnstileSiteKey:      a.sysValue("auth.turnstile_site_key"),
		Maintenance:           a.sysOn("app.maintenance", false),
	})
}

func (a *App) handleGoogleAuth(c *gin.Context) {
	var req googleAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, CodeInvalidBody, "invalid request body")
		return
	}
	if a.rejectWebMaintenance(c, req.Client) {
		return
	}
	setLoginKind(c, loginClientKind(req.Client))
	if !a.googleEnabled() {
		fail(c, http.StatusServiceUnavailable, CodeGoogleDisabled, "google sign-in is not enabled")
		return
	}
	ip := c.ClientIP()
	if !a.LoginGuard.AllowIP(ip) {
		a.recordLoginLog(c, "google", "failed", "ip rate limited")
		fail(c, http.StatusTooManyRequests, CodeLoginRateLimited, "too many login attempts from this ip")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()
	ident, err := a.googleVerifier().Verify(ctx, req.IDToken, a.googleClientID())
	if err != nil {
		a.recordLoginLog(c, "google", "failed", "invalid google token")
		fail(c, http.StatusUnauthorized, CodeGoogleTokenInvalid, "google sign-in failed")
		return
	}
	if ident.Email == "" || !ident.EmailVerified {
		a.recordLoginLog(c, ident.Email, "failed", "google email unverified")
		fail(c, http.StatusUnauthorized, CodeGoogleEmailUnverified, "google email is not verified")
		return
	}

	kind := models.UserKindWeb
	if strings.EqualFold(strings.TrimSpace(req.Client), "admin") {
		kind = models.UserKindAdmin
	}

	user, created, err := a.findOrCreateGoogleUser(ident, kind)
	if err != nil {
		switch err {
		case errGoogleRegisterDisabled:
			a.recordLoginLog(c, ident.Email, "failed", "google register disabled")
			fail(c, http.StatusForbidden, CodeGoogleRegisterDisabled, "google registration is disabled")
		case errGoogleAccountConflict:
			a.recordLoginLog(c, ident.Email, "failed", "google account conflict")
			fail(c, http.StatusForbidden, CodeGoogleAccountConflict, "this google account is already linked to another user")
		case errGoogleEmailTaken:
			a.recordLoginLog(c, ident.Email, "failed", "google email taken")
			fail(c, http.StatusConflict, CodeEmailExists, "email already exists")
		default:
			a.recordLoginLog(c, ident.Email, "failed", "google user create")
			fail(c, http.StatusInternalServerError, CodeAssignRoles, "failed to create user")
		}
		return
	}
	if err := a.loadAccount(kind, &user, user.ID); err != nil {
		fail(c, http.StatusInternalServerError, CodeAssignRoles, "failed to create user")
		return
	}

	now := time.Now()
	if security.IsLocked(user.LockedUntil, now) {
		a.recordLoginLog(c, user.Username, "failed", "account locked")
		fail(c, http.StatusForbidden, CodeAccountLocked, "account locked")
		return
	}
	if user.Status != "active" {
		a.recordLoginLog(c, user.Username, "failed", "inactive")
		fail(c, http.StatusUnauthorized, CodeBadCredentials, "invalid credentials")
		return
	}

	reason := "google"
	if created {
		reason = "google-register"
	}
	a.finishLogin(c, user, ip, reason)
}

var (
	errGoogleRegisterDisabled = errString("google register disabled")
	errGoogleAccountConflict  = errString("google account conflict")
	errGoogleEmailTaken       = errString("google email taken")
)

func (a *App) findOrCreateGoogleUser(ident googleid.Identity, kind string) (models.User, bool, error) {
	kind = models.NormalizeUserKind(kind)
	ident.Email = normalizeEmail(ident.Email)
	var byGoogle models.User
	if ident.Subject != "" {
		err := a.accounts(kind).Where("google_id = ? AND google_id <> ''", ident.Subject).First(&byGoogle).Error
		if err == nil {
			byGoogle.Kind = kind
			updates := map[string]any{}
			if byGoogle.Email == "" && ident.Email != "" && !a.emailTaken(kind, ident.Email, byGoogle.ID) {
				updates["email"] = ident.Email
				if ident.EmailVerified {
					updates["email_verified"] = true
				}
			}
			if byGoogle.Nickname == "" && ident.Name != "" {
				updates["nickname"] = ident.Name
			}
			if byGoogle.Avatar == "" && ident.Picture != "" && len(ident.Picture) <= 255 {
				updates["avatar"] = ident.Picture
			}
			if len(updates) > 0 {
				_ = a.updateAccount(&byGoogle, updates)
			}
			return byGoogle, false, nil
		}
		if err != nil && err != gorm.ErrRecordNotFound {
			return models.User{}, false, err
		}
	}

	var byEmail models.User
	err := a.accounts(kind).Where("lower(email) = ? AND email <> ''", ident.Email).First(&byEmail).Error
	if err == nil {
		byEmail.Kind = kind
		if byEmail.GoogleID != "" && byEmail.GoogleID != ident.Subject {
			return models.User{}, false, errGoogleAccountConflict
		}
		updates := map[string]any{"google_id": ident.Subject}
		if byEmail.Nickname == "" && ident.Name != "" {
			updates["nickname"] = ident.Name
		}
		if byEmail.Avatar == "" && ident.Picture != "" && len(ident.Picture) <= 255 {
			updates["avatar"] = ident.Picture
		}
		if err := a.updateAccount(&byEmail, updates); err != nil {
			return models.User{}, false, err
		}
		byEmail.GoogleID = ident.Subject
		return byEmail, false, nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return models.User{}, false, err
	}

	if kind != models.UserKindWeb || !a.googleRegisterEnabled() {
		return models.User{}, false, errGoogleRegisterDisabled
	}
	if ident.Email != "" && a.emailTaken(kind, ident.Email, 0) {
		return models.User{}, false, errGoogleEmailTaken
	}

	username, err := a.uniqueUsername(kind, googleUsername(ident.Email, ident.Subject))
	if err != nil {
		return models.User{}, false, err
	}
	hash, err := randomUnusableHash()
	if err != nil {
		return models.User{}, false, err
	}
	roles, err := a.defaultRolesForKind(models.UserKindWeb, []models.Role{})
	if err != nil {
		return models.User{}, false, err
	}
	user := models.User{
		Username:        username,
		PasswordHash:    hash,
		Nickname:        ident.Name,
		Avatar:          clip(ident.Picture, 255),
		Email:           ident.Email,
		Status:          "active",
		Timezone:        mailer.DefaultTimezone,
		MarketingOptIn:  false,
		EmailVerified:   true,
		Kind:            models.UserKindWeb,
		GoogleID:        ident.Subject,
		MustSetPassword: true,
	}
	var created bool
	for attempt := 0; attempt < 3 && !created; attempt++ {
		if attempt > 0 {
			next, err := a.uniqueUsername(kind, googleUsername(ident.Email, ident.Subject))
			if err != nil {
				return models.User{}, false, err
			}
			user.Username = next
		}
		err := a.withTx(func(tx *gorm.DB) error {
			if err := models.Accounts(tx, user.Kind).Create(&user).Error; err != nil {
				return err
			}
			return models.ReplaceUserRoles(tx, user.Kind, user.ID, roles)
		})
		if err == nil {
			created = true
			break
		}
		if !isUniqueViolation(err) {
			return models.User{}, false, err
		}
	}
	if !created {
		return models.User{}, false, gorm.ErrDuplicatedKey
	}
	if err := seed.SyncUserRoles(a.Enforcer, seed.CasbinSub(user.Kind, user.ID), roles); err != nil {
		return models.User{}, false, err
	}
	user.Roles = roles
	return user, true, nil
}

func googleUsername(email, sub string) string {
	local := email
	if i := strings.IndexByte(email, '@'); i > 0 {
		local = email[:i]
	}
	var b strings.Builder
	for _, r := range strings.ToLower(local) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	base := strings.Trim(b.String(), ".-_")
	if base == "" {
		base = "g" + strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				return r
			}
			return -1
		}, sub)
	}
	if base == "" || base == "g" {
		base = "google"
	}
	if len(base) > 48 {
		base = base[:48]
	}
	return base
}

func (a *App) uniqueUsername(kind, base string) (string, error) {
	candidate := base
	for i := 0; i < 30; i++ {
		var n int64
		if err := a.accounts(kind).Where("username = ?", candidate).Count(&n).Error; err != nil {
			return "", err
		}
		if n == 0 {
			return candidate, nil
		}
		suffix := fmt.Sprintf("%d", i+2)
		keep := 64 - len(suffix)
		if keep < 1 {
			keep = 1
		}
		stem := base
		if len(stem) > keep {
			stem = stem[:keep]
		}
		candidate = stem + suffix
	}
	buf := make([]byte, 3)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return clipLen(base, 56) + hex.EncodeToString(buf), nil
}

func clipLen(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return ""
}

func randomUnusableHash() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return passwd.Hash(hex.EncodeToString(buf))
}
