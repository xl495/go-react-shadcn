package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/models"
)

func (a *App) handleDashboard(c *gin.Context) {
	var users, roles, perms, dicts, configs, logs int64
	if err := a.DB.Model(&models.User{}).Count(&users).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50040, "failed to load stats")
		return
	}
	if err := a.DB.Model(&models.Role{}).Count(&roles).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50040, "failed to load stats")
		return
	}
	if err := a.DB.Model(&models.Permission{}).Count(&perms).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50040, "failed to load stats")
		return
	}
	_ = a.DB.Model(&models.DictType{}).Count(&dicts).Error
	_ = a.DB.Model(&models.SysConfig{}).Count(&configs).Error
	_ = a.DB.Model(&models.OpLog{}).Count(&logs).Error
	ok(c, gin.H{
		"users":       users,
		"roles":       roles,
		"permissions": perms,
		"dicts":       dicts,
		"configs":     configs,
		"logs":        logs,
	})
}
