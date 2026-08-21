package httpserver

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/mailer"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/secretbox"
	"gorm.io/gorm"
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
	total, okCount := countOrFail(c, q, CodeListConfigs, "failed to list configs")
	if !okCount {
		return
	}
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
	if a.rejectDisabledCaptcha(c, req.Key, req.Value) {
		return
	}
	if req.Group == "" {
		req.Group = "app"
	}
	row := models.SysConfig{Key: req.Key, Value: req.Value, Name: req.Name, Group: req.Group, Remark: req.Remark}
	if err := a.sealConfigValue(&row); err != nil {
		fail(c, http.StatusInternalServerError, 50071, "failed to store config")
		return
	}
	if err := a.DB.Create(&row).Error; err != nil {
		fail(c, http.StatusConflict, 40970, "config key already exists")
		return
	}
	a.syscfg.invalidate()
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
	if a.rejectDisabledCaptcha(c, row.Key, req.Value) {
		return
	}
	if req.Name != "" {
		row.Name = req.Name
	}
	row.Value = mailer.KeepSecret(row.Key, req.Value, row.Value)
	if mailer.LooksSecret(row.Key) && req.Value != "" && req.Value != mailer.SecretMask {
		if err := a.sealConfigValue(&row); err != nil {
			fail(c, http.StatusInternalServerError, 50071, "failed to store config")
			return
		}
	}
	if req.Group != "" {
		row.Group = req.Group
	}
	row.Remark = req.Remark
	if err := a.DB.Save(&row).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50071, "failed to update config")
		return
	}
	a.syscfg.invalidate()
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
	a.syscfg.invalidate()
	ok(c, gin.H{"deleted": row.ID})
}

type batchConfigItem struct {
	ID     uint   `json:"id"`
	Key    string `json:"key"`
	Value  string `json:"value"`
	Name   string `json:"name"`
	Group  string `json:"group"`
	Remark string `json:"remark"`
}

func (a *App) handleBatchConfigs(c *gin.Context) {
	var req struct {
		Items []batchConfigItem `json:"items"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Items) == 0 {
		fail(c, http.StatusBadRequest, CodeInvalidConfigBody, "invalid request body")
		return
	}
	for _, item := range req.Items {
		key := item.Key
		if item.ID != 0 && key == "" {
			var row models.SysConfig
			if err := a.DB.Select("key").First(&row, item.ID).Error; err != nil {
				fail(c, http.StatusNotFound, CodeConfigNotFound, "config not found")
				return
			}
			key = row.Key
		}
		if a.rejectDisabledCaptcha(c, key, item.Value) {
			return
		}
	}
	err := a.DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range req.Items {
			if item.ID == 0 {
				row := models.SysConfig{Key: item.Key, Value: item.Value, Name: item.Name, Group: item.Group, Remark: item.Remark}
				if row.Group == "" {
					row.Group = "app"
				}
				if err := a.sealConfigValue(&row); err != nil {
					return err
				}
				if err := tx.Create(&row).Error; err != nil {
					return err
				}
				continue
			}
			var row models.SysConfig
			if err := tx.First(&row, item.ID).Error; err != nil {
				return err
			}
			if item.Name != "" {
				row.Name = item.Name
			}
			row.Value = mailer.KeepSecret(row.Key, item.Value, row.Value)
			if mailer.LooksSecret(row.Key) && item.Value != "" && item.Value != mailer.SecretMask {
				if err := a.sealConfigValue(&row); err != nil {
					return err
				}
			}
			if item.Group != "" {
				row.Group = item.Group
			}
			row.Remark = item.Remark
			if err := tx.Save(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, CodeUpdateConfig, "failed to update config")
		return
	}
	a.syscfg.invalidate()
	ok(c, gin.H{"saved": len(req.Items)})
}

func presentConfigs(rows []models.SysConfig) []models.SysConfig {
	out := make([]models.SysConfig, len(rows))
	for i, row := range rows {
		out[i] = mailer.Redact(row)
	}
	return out
}

func (a *App) sealConfigValue(row *models.SysConfig) error {
	if row == nil || !mailer.LooksSecret(row.Key) || row.Value == "" || row.Value == mailer.SecretMask {
		return nil
	}
	sealed, err := secretbox.Seal(a.Cfg.JWTSecret, row.Value)
	if err != nil {
		return err
	}
	row.Value = sealed
	return nil
}

func (a *App) rejectDisabledCaptcha(c *gin.Context, key, value string) bool {
	if a.Cfg.DevMode {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(key), "auth.captcha_provider") {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(value), "none") {
		return false
	}
	fail(c, http.StatusBadRequest, CodeInvalidConfigBody, "captcha cannot be disabled in production")
	return true
}
