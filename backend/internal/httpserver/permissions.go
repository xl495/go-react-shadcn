package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/seed"
)

type permissionRequest struct {
	Name        string `json:"name"`
	Code        string `json:"code"`
	Path        string `json:"path"`
	Method      string `json:"method"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
}

func normalizeKind(k string) string {
	switch k {
	case seed.KindMenu, seed.KindButton, seed.KindAPI:
		return k
	default:
		return seed.KindButton
	}
}

func (a *App) handleListPermissions(c *gin.Context) {
	var perms []models.Permission
	if err := a.DB.Order("id asc").Find(&perms).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50030, "failed to list permissions")
		return
	}
	out := make([]permissionDTO, 0, len(perms))
	for _, p := range perms {
		out = append(out, toPermissionDTO(p))
	}
	ok(c, out)
}

func (a *App) handleCreatePermission(c *gin.Context) {
	var req permissionRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" || req.Code == "" || req.Path == "" || req.Method == "" {
		fail(c, http.StatusBadRequest, 40030, "name, code, path and method required")
		return
	}
	perm := models.Permission{
		Name:        req.Name,
		Code:        req.Code,
		Path:        req.Path,
		Method:      normalizeMethod(req.Method),
		Kind:        normalizeKind(req.Kind),
		Description: req.Description,
	}
	if err := a.DB.Create(&perm).Error; err != nil {
		fail(c, http.StatusConflict, 40930, "permission code already exists")
		return
	}
	ok(c, toPermissionDTO(perm))
}

func (a *App) handleUpdatePermission(c *gin.Context) {
	var perm models.Permission
	if err := a.DB.First(&perm, c.Param("id")).Error; err != nil {
		fail(c, http.StatusNotFound, 40430, "permission not found")
		return
	}
	var req permissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, 40031, "invalid request body")
		return
	}
	if req.Name != "" {
		perm.Name = req.Name
	}
	if req.Path != "" {
		perm.Path = req.Path
	}
	if req.Method != "" {
		perm.Method = normalizeMethod(req.Method)
	}
	if req.Kind != "" {
		perm.Kind = normalizeKind(req.Kind)
	}
	if req.Description != "" || req.Name != "" {
		perm.Description = req.Description
	}
	if err := a.DB.Save(&perm).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50031, "failed to update permission")
		return
	}
	if err := a.resyncRolesUsing(perm.ID); err != nil {
		fail(c, http.StatusInternalServerError, 50022, "failed to sync rbac")
		return
	}
	ok(c, toPermissionDTO(perm))
}

func (a *App) handleDeletePermission(c *gin.Context) {
	var perm models.Permission
	if err := a.DB.First(&perm, c.Param("id")).Error; err != nil {
		fail(c, http.StatusNotFound, 40430, "permission not found")
		return
	}
	if err := a.DB.Model(&perm).Association("Roles").Clear(); err != nil {
		fail(c, http.StatusInternalServerError, 50032, "failed to detach permission")
		return
	}
	if err := a.DB.Delete(&perm).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50033, "failed to delete permission")
		return
	}
	var remaining []models.Role
	if err := a.DB.Preload("Permissions").Find(&remaining).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50022, "failed to sync rbac")
		return
	}
	for _, role := range remaining {
		if err := seed.SyncRolePolicies(a.Enforcer, role.Code, role.Permissions); err != nil {
			fail(c, http.StatusInternalServerError, 50022, "failed to sync rbac")
			return
		}
	}
	ok(c, gin.H{"deleted": perm.ID})
}

func (a *App) resyncRolesUsing(permissionID uint) error {
	var roles []models.Role
	if err := a.DB.Preload("Permissions").
		Joins("JOIN role_permissions rp ON rp.role_id = roles.id").
		Where("rp.permission_id = ?", permissionID).
		Find(&roles).Error; err != nil {
		return err
	}
	for _, role := range roles {
		if err := seed.SyncRolePolicies(a.Enforcer, role.Code, role.Permissions); err != nil {
			return err
		}
	}
	return nil
}
