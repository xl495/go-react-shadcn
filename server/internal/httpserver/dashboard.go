package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/models"
)

func (a *App) handleDashboard(c *gin.Context) {
	var stats struct {
		Users      int64 `gorm:"column:users"`
		Roles      int64 `gorm:"column:roles"`
		Perms      int64 `gorm:"column:permissions"`
		Dicts      int64 `gorm:"column:dicts"`
		Cfgs       int64 `gorm:"column:configs"`
		Logs       int64 `gorm:"column:logs"`
		MailQueued int64 `gorm:"column:mail_queued"`
	}
	err := a.DB.Raw(`SELECT
		(SELECT COUNT(*) FROM admin_user) + (SELECT COUNT(*) FROM web_user) AS users,
		(SELECT COUNT(*) FROM roles) AS roles,
		(SELECT COUNT(*) FROM permissions) AS permissions,
		(SELECT COUNT(*) FROM dict_types) AS dicts,
		(SELECT COUNT(*) FROM sys_configs) AS configs,
		(SELECT COUNT(*) FROM op_logs) AS logs,
		(SELECT COUNT(*) FROM mail_jobs WHERE status IN ('queued','sending')) AS mail_queued`).Scan(&stats).Error
	if err != nil {
		fail(c, http.StatusInternalServerError, 50040, "failed to load stats")
		return
	}
	var recent []models.LoginLog
	_ = a.DB.Order("id desc").Limit(8).Find(&recent).Error
	var failed []models.LoginLog
	_ = a.DB.Where("status = ?", "failed").Order("id desc").Limit(5).Find(&failed).Error
	ok(c, gin.H{
		"users":        stats.Users,
		"roles":        stats.Roles,
		"permissions":  stats.Perms,
		"dicts":        stats.Dicts,
		"configs":      stats.Cfgs,
		"logs":         stats.Logs,
		"mailQueued":   stats.MailQueued,
		"recentLogins": recent,
		"failedLogins": failed,
	})
}
