package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
	var rows []models.SysConfig
	q := a.DB.Order("id asc")
	if g := c.Query("group"); g != "" {
		q = q.Where("`group` = ?", g)
	}
	if err := q.Find(&rows).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50070, "failed to list configs")
		return
	}
	ok(c, rows)
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
	ok(c, row)
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
	row.Value = req.Value
	if req.Group != "" {
		row.Group = req.Group
	}
	row.Remark = req.Remark
	if err := a.DB.Save(&row).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50071, "failed to update config")
		return
	}
	ok(c, row)
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
