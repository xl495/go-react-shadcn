package httpserver

import (
	"strings"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/token"
	"gorm.io/gorm"
)

func loginClientKind(client string) string {
	if strings.EqualFold(strings.TrimSpace(client), "web") {
		return models.UserKindWeb
	}
	return models.UserKindAdmin
}

func queryUserKind(c *gin.Context) string {
	kind := strings.TrimSpace(c.Query("kind"))
	if kind == "" {
		return models.UserKindAdmin
	}
	return models.NormalizeUserKind(kind)
}

func (a *App) accounts(kind string) *gorm.DB {
	return models.Accounts(a.DB, kind)
}

func (a *App) loadAccount(kind string, dest *models.User, conds ...any) error {
	return models.LoadAccount(a.DB, kind, dest, conds...)
}

func (a *App) saveAccount(user *models.User) error {
	return a.accounts(user.Kind).Omit("Roles", "Dept").Save(user).Error
}

func (a *App) updateAccount(user *models.User, values map[string]any) error {
	return a.accounts(user.Kind).Where("id = ?", user.ID).Updates(values).Error
}

func (a *App) currentAccount(c *gin.Context) (models.User, *token.Claims, error) {
	claims := currentUser(c)
	var user models.User
	if claims == nil {
		return user, nil, gorm.ErrRecordNotFound
	}
	err := a.loadAccount(claimsKind(claims), &user, claims.UserID)
	return user, claims, err
}
