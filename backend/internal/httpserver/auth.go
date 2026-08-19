package httpserver

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/passwd"
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
	if req.CaptchaID == "" || req.CaptchaCode == "" {
		fail(c, http.StatusBadRequest, 40002, "captcha required")
		return
	}
	if !a.Captcha.Verify(req.CaptchaID, req.CaptchaCode) {
		a.recordLog(c, "auth", "login", "invalid captcha")
		fail(c, http.StatusBadRequest, 40003, "invalid captcha")
		return
	}
	if req.Username == "" || req.Password == "" {
		fail(c, http.StatusUnauthorized, 40103, "invalid credentials")
		return
	}

	var user models.User
	if err := a.DB.Preload("Roles.Permissions").Where("username = ?", req.Username).First(&user).Error; err != nil {
		fail(c, http.StatusUnauthorized, 40103, "invalid credentials")
		return
	}
	if user.Status != "active" || !passwd.Match(user.PasswordHash, req.Password) {
		a.recordLog(c, "auth", "login", "invalid credentials:"+req.Username)
		fail(c, http.StatusUnauthorized, 40103, "invalid credentials")
		return
	}

	now := time.Now()
	user.LastLoginAt = &now
	user.LastLoginIP = c.ClientIP()
	_ = a.DB.Model(&user).Updates(map[string]any{
		"last_login_at": user.LastLoginAt,
		"last_login_ip": user.LastLoginIP,
	}).Error

	roles := roleCodes(user.Roles)
	tok, exp, err := a.Tokens.Issue(user.ID, user.Username, roles)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50003, "failed to issue token")
		return
	}
	ok(c, loginData{
		Token:     tok,
		ExpiresAt: exp,
		User:      toUserDTO(user),
	})
	a.recordLog(c, "auth", "login", "ok:"+user.Username)
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
	if !passwd.Match(user.PasswordHash, req.OldPassword) {
		fail(c, http.StatusBadRequest, 40041, "current password is wrong")
		return
	}
	hash, err := passwd.Hash(req.NewPassword)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50011, "failed to hash password")
		return
	}
	if err := a.DB.Model(&user).Update("password_hash", hash).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50042, "failed to change password")
		return
	}
	ok(c, gin.H{"changed": true})
}
