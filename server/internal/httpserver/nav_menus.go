package httpserver

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/models"
	"gorm.io/gorm"
)

type navMenuNode struct {
	ID        uint          `json:"id"`
	ParentID  *uint         `json:"parentId,omitempty"`
	Audience  string        `json:"audience"`
	Name      string        `json:"name"`
	Code      string        `json:"code"`
	RoutePath string        `json:"routePath"`
	Component string        `json:"component"`
	Icon      string        `json:"icon"`
	Sort      int           `json:"sort"`
	Hidden    bool          `json:"hidden"`
	PermCode  string        `json:"permCode"`
	Status    string        `json:"status"`
	IsSystem  bool          `json:"isSystem"`
	Children  []navMenuNode `json:"children,omitempty"`
}

type navMenuRequest struct {
	ParentID  *uint  `json:"parentId"`
	Audience  string `json:"audience"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	RoutePath string `json:"routePath"`
	Component string `json:"component"`
	Icon      string `json:"icon"`
	Sort      *int   `json:"sort"`
	Hidden    *bool  `json:"hidden"`
	PermCode  string `json:"permCode"`
	Status    string `json:"status"`
}

type navReorderRequest struct {
	Items []struct {
		ID       uint  `json:"id"`
		Sort     int   `json:"sort"`
		ParentID *uint `json:"parentId"`
	} `json:"items"`
}

func (a *App) handleListNavMenus(c *gin.Context) {
	audience := strings.TrimSpace(c.Query("audience"))
	if audience == "" {
		audience = models.NavAudienceAdmin
	}
	if audience != models.NavAudienceAdmin && audience != models.NavAudienceWeb {
		fail(c, http.StatusBadRequest, CodeInvalidBody, "invalid audience")
		return
	}
	var rows []models.NavMenu
	if err := a.DB.Where("audience = ?", audience).Order("sort asc, id asc").Find(&rows).Error; err != nil {
		fail(c, http.StatusInternalServerError, CodeListMenus, "failed to list menus")
		return
	}
	ok(c, buildNavAdminTree(rows, nil))
}

func (a *App) handleCreateNavMenu(c *gin.Context) {
	var req navMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Code) == "" {
		fail(c, http.StatusBadRequest, CodeInvalidBody, "name and code required")
		return
	}
	audience := strings.TrimSpace(req.Audience)
	if audience == "" {
		audience = models.NavAudienceAdmin
	}
	if audience != models.NavAudienceAdmin && audience != models.NavAudienceWeb {
		fail(c, http.StatusBadRequest, CodeInvalidBody, "invalid audience")
		return
	}
	sort := 0
	if req.Sort != nil {
		sort = *req.Sort
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "active"
	}
	row := models.NavMenu{
		ParentID: req.ParentID, Audience: audience, Name: strings.TrimSpace(req.Name),
		Code: strings.TrimSpace(req.Code), RoutePath: strings.TrimSpace(req.RoutePath),
		Component: strings.TrimSpace(req.Component), Icon: strings.TrimSpace(req.Icon),
		Sort: sort, PermCode: strings.TrimSpace(req.PermCode), Status: status,
	}
	if req.Hidden != nil {
		row.Hidden = *req.Hidden
	}
	if err := a.DB.Create(&row).Error; err != nil {
		fail(c, http.StatusConflict, 40993, "menu code already exists")
		return
	}
	ok(c, row)
}

func (a *App) handleUpdateNavMenu(c *gin.Context) {
	var row models.NavMenu
	if err := a.DB.First(&row, c.Param("id")).Error; err != nil {
		fail(c, http.StatusNotFound, CodeNavMenuNotFound, "menu not found")
		return
	}
	var req navMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, CodeInvalidBody, "invalid request body")
		return
	}
	if req.Name != "" {
		row.Name = strings.TrimSpace(req.Name)
	}
	if req.RoutePath != "" || req.Component != "" || req.Icon != "" || req.PermCode != "" {
		if req.RoutePath != "" {
			row.RoutePath = strings.TrimSpace(req.RoutePath)
		}
		if req.Component != "" {
			row.Component = strings.TrimSpace(req.Component)
		}
		if req.Icon != "" {
			row.Icon = strings.TrimSpace(req.Icon)
		}
		if req.PermCode != "" {
			row.PermCode = strings.TrimSpace(req.PermCode)
		}
	}
	if req.Sort != nil {
		row.Sort = *req.Sort
	}
	if req.Hidden != nil {
		row.Hidden = *req.Hidden
	}
	if req.Status != "" {
		row.Status = strings.TrimSpace(req.Status)
	}
	row.ParentID = req.ParentID
	if err := a.DB.Save(&row).Error; err != nil {
		fail(c, http.StatusInternalServerError, CodeListMenus, "failed to update menu")
		return
	}
	ok(c, row)
}

func (a *App) handleDeleteNavMenu(c *gin.Context) {
	var row models.NavMenu
	if err := a.DB.First(&row, c.Param("id")).Error; err != nil {
		fail(c, http.StatusNotFound, CodeNavMenuNotFound, "menu not found")
		return
	}
	if row.IsSystem {
		fail(c, http.StatusBadRequest, CodeCannotDeleteSystemMenu, "cannot delete system menu")
		return
	}
	var n int64
	if err := a.DB.Model(&models.NavMenu{}).Where("parent_id = ?", row.ID).Count(&n).Error; err != nil {
		fail(c, http.StatusInternalServerError, CodeListMenus, "failed to delete menu")
		return
	}
	if n > 0 {
		fail(c, http.StatusBadRequest, CodeNavMenuHasChildren, "menu has children")
		return
	}
	if err := a.DB.Delete(&row).Error; err != nil {
		fail(c, http.StatusInternalServerError, CodeListMenus, "failed to delete menu")
		return
	}
	ok(c, gin.H{"deleted": row.ID})
}

func (a *App) handleReorderNavMenus(c *gin.Context) {
	var req navReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Items) == 0 {
		fail(c, http.StatusBadRequest, CodeInvalidBody, "invalid request body")
		return
	}
	err := a.DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range req.Items {
			if err := tx.Model(&models.NavMenu{}).Where("id = ?", item.ID).Updates(map[string]any{
				"sort":      item.Sort,
				"parent_id": item.ParentID,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, CodeListMenus, "failed to reorder menus")
		return
	}
	ok(c, gin.H{"updated": len(req.Items)})
}

func buildNavAdminTree(rows []models.NavMenu, parentID *uint) []navMenuNode {
	out := make([]navMenuNode, 0)
	for _, p := range rows {
		same := (parentID == nil && p.ParentID == nil) ||
			(parentID != nil && p.ParentID != nil && *parentID == *p.ParentID)
		if !same {
			continue
		}
		out = append(out, navMenuNode{
			ID: p.ID, ParentID: p.ParentID, Audience: p.Audience, Name: p.Name, Code: p.Code,
			RoutePath: p.RoutePath, Component: p.Component, Icon: p.Icon, Sort: p.Sort,
			Hidden: p.Hidden, PermCode: p.PermCode, Status: p.Status, IsSystem: p.IsSystem,
			Children: buildNavAdminTree(rows, &p.ID),
		})
	}
	return out
}
