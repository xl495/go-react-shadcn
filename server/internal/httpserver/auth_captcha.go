package httpserver

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/googleid"
	"go-react-shadcn/internal/siteverify"
)

type challengeInput struct {
	CaptchaID      string `json:"captchaId"`
	CaptchaCode    string `json:"captchaCode"`
	CaptchaToken   string `json:"captchaToken"`
	CaptchaVersion string `json:"captchaVersion"`
}

func (a *App) siteChecker() siteverifyClient {
	if a.SiteVerify != nil {
		return a.SiteVerify
	}
	return &siteverify.Client{HTTP: a.httpClient()}
}

func (a *App) googleVerifier() googleid.Verifier {
	if a.GoogleVerify != nil {
		return a.GoogleVerify
	}
	return googleid.HTTPVerifier{Client: a.httpClient()}
}

type siteverifyClient interface {
	Check(ctx context.Context, endpoint, secret, response, remoteIP string) (siteverify.Result, error)
}

func (a *App) requireCaptcha(c *gin.Context, in challengeInput, action string) bool {
	switch a.captchaProvider() {
	case "none":
		return true
	case "image":
		if in.CaptchaID == "" || in.CaptchaCode == "" {
			fail(c, http.StatusBadRequest, CodeCaptchaRequired, "captcha required")
			return false
		}
		if !a.Captcha.Verify(in.CaptchaID, in.CaptchaCode) {
			fail(c, http.StatusBadRequest, CodeInvalidCaptcha, "invalid captcha")
			return false
		}
		return true
	case "recaptcha":
		return a.verifyRecaptcha(c, in, action)
	case "turnstile":
		return a.verifyTurnstile(c, in)
	default:
		return true
	}
}

func (a *App) verifyRecaptcha(c *gin.Context, in challengeInput, action string) bool {
	token := strings.TrimSpace(in.CaptchaToken)
	if token == "" {
		fail(c, http.StatusBadRequest, CodeCaptchaRequired, "captcha required")
		return false
	}
	version := strings.ToLower(strings.TrimSpace(in.CaptchaVersion))
	if version == "" {
		version = "v3"
	}
	secretV3 := a.sysValue("auth.recaptcha_secret_v3")
	secretV2 := a.sysValue("auth.recaptcha_secret_v2")
	if version == "v2" {
		if secretV2 == "" {
			fail(c, http.StatusBadRequest, CodeCaptchaUnavailable, "captcha is not configured")
			return false
		}
		return a.checkRemoteCaptcha(c, siteverify.Recaptcha, secretV2, token, "")
	}
	if secretV3 == "" {
		if secretV2 != "" {
			fail(c, http.StatusBadRequest, CodeCaptchaFallback, "complete the extra captcha check")
			return false
		}
		fail(c, http.StatusBadRequest, CodeCaptchaUnavailable, "captcha is not configured")
		return false
	}
	result, ok := a.remoteCaptchaResult(c, siteverify.Recaptcha, secretV3, token)
	if !ok {
		return false
	}
	if action != "" && result.Action != "" && !strings.EqualFold(result.Action, action) {
		fail(c, http.StatusBadRequest, CodeInvalidCaptcha, "invalid captcha")
		return false
	}
	if result.Score < a.recaptchaMinScore() {
		if secretV2 != "" && a.sysValue("auth.recaptcha_site_key_v2") != "" {
			fail(c, http.StatusBadRequest, CodeCaptchaFallback, "complete the extra captcha check")
			return false
		}
		fail(c, http.StatusBadRequest, CodeInvalidCaptcha, "invalid captcha")
		return false
	}
	return true
}

func (a *App) verifyTurnstile(c *gin.Context, in challengeInput) bool {
	token := strings.TrimSpace(in.CaptchaToken)
	if token == "" {
		fail(c, http.StatusBadRequest, CodeCaptchaRequired, "captcha required")
		return false
	}
	secret := a.sysValue("auth.turnstile_secret")
	if secret == "" {
		fail(c, http.StatusBadRequest, CodeCaptchaUnavailable, "captcha is not configured")
		return false
	}
	return a.checkRemoteCaptcha(c, siteverify.Turnstile, secret, token, "")
}

func (a *App) checkRemoteCaptcha(c *gin.Context, endpoint, secret, token, expectedAction string) bool {
	result, ok := a.remoteCaptchaResult(c, endpoint, secret, token)
	if !ok {
		return false
	}
	if expectedAction != "" && result.Action != "" && !strings.EqualFold(result.Action, expectedAction) {
		fail(c, http.StatusBadRequest, CodeInvalidCaptcha, "invalid captcha")
		return false
	}
	return true
}

func (a *App) remoteCaptchaResult(c *gin.Context, endpoint, secret, token string) (siteverify.Result, bool) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()
	result, err := a.siteChecker().Check(ctx, endpoint, secret, token, c.ClientIP())
	if err != nil {
		fail(c, http.StatusBadRequest, CodeInvalidCaptcha, "invalid captcha")
		return siteverify.Result{}, false
	}
	if !result.Success {
		fail(c, http.StatusBadRequest, CodeInvalidCaptcha, "invalid captcha")
		return siteverify.Result{}, false
	}
	return result, true
}
