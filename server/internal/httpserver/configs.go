package httpserver

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/mailer"
	"go-react-shadcn/internal/models"
)

type configRequest struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Name   string `json:"name"`
	Group  string `json:"group"`
	Remark string `json:"remark"`
}

func (a *App) handleListConfigs(c *gin.Context) {
	p := parsePage(c, 50, 500)
	q := a.DB.Model(&models.SysConfig{})
	if g := strings.TrimSpace(c.Query("group")); g != "" {
		q = q.Where(`"group" = ?`, g)
	}
	q = applyContains(q, c.Query("q"), `"key"`, "name", "remark")
	var total int64
	_ = q.Count(&total).Error
	var rows []models.SysConfig
	if err := q.Order("id asc").Offset(p.Offset()).Limit(p.PageSize).Find(&rows).Error; err != nil {
		fail(c, http.StatusInternalServerError, CodeListConfigs, "failed to list configs")
		return
	}
	ok(c, pageResult[models.SysConfig]{Items: presentConfigs(rows), Total: total, Page: p.Page, PageSize: p.PageSize})
}

func (a *App) handleCreateConfig(c *gin.Context) {
	var req configRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Key == "" || req.Name == "" {
		fail(c, http.StatusBadRequest, 40070, "key and name required")
		return
	}
	if req.Group == "" {
		req.Group = "app"
	}
	row := models.SysConfig{Key: req.Key, Value: req.Value, Name: req.Name, Group: req.Group, Remark: req.Remark}
	if err := a.DB.Create(&row).Error; err != nil {
		fail(c, http.StatusConflict, 40970, "config key already exists")
		return
	}
	ok(c, mailer.Redact(row))
}

func (a *App) handleUpdateConfig(c *gin.Context) {
	var row models.SysConfig
	if err := a.DB.First(&row, c.Param("id")).Error; err != nil {
		fail(c, http.StatusNotFound, 40470, "config not found")
		return
	}
	var req configRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, 40071, "invalid request body")
		return
	}
	if req.Name != "" {
		row.Name = req.Name
	}
	row.Value = mailer.KeepSecret(row.Key, req.Value, row.Value)
	if req.Group != "" {
		row.Group = req.Group
	}
	row.Remark = req.Remark
	if err := a.DB.Save(&row).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50071, "failed to update config")
		return
	}
	ok(c, mailer.Redact(row))
}

func (a *App) handleDeleteConfig(c *gin.Context) {
	var row models.SysConfig
	if err := a.DB.First(&row, c.Param("id")).Error; err != nil {
		fail(c, http.StatusNotFound, 40470, "config not found")
		return
	}
	if err := a.DB.Delete(&row).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50072, "failed to delete config")
		return
	}
	ok(c, gin.H{"deleted": row.ID})
}

func presentConfigs(rows []models.SysConfig) []models.SysConfig {
	out := make([]models.SysConfig, len(rows))
	for i, row := range rows {
		out[i] = mailer.Redact(row)
	}
	return out
}
