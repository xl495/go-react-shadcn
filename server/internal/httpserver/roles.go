package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/seed"
	"gorm.io/gorm"
)

type createRoleRequest struct {
	Name          string `json:"name"`
	Code          string `json:"code"`
	Description   string `json:"description"`
	DataScope     string `json:"dataScope"`
	PermissionIDs []uint `json:"permissionIds"`
}

type updateRoleRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	DataScope   *string `json:"dataScope"`
}

type assignPermissionsRequest struct {
	PermissionIDs []uint `json:"permissionIds"`
}

func (a *App) handleListRoles(c *gin.Context) {
	p := parsePage(c, 50, 500)
	q := a.DB.Model(&models.Role{})
	var total int64
	_ = q.Count(&total).Error
	var roles []models.Role
	if err := q.Preload("Permissions").Order("id asc").Offset(p.Offset()).Limit(p.PageSize).Find(&roles).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50020, "failed to list roles")
		return
	}
	out := make([]roleDTO, 0, len(roles))
	for _, r := range roles {
		out = append(out, toRoleDTO(r, true))
	}
	ok(c, pageResult[roleDTO]{Items: out, Total: total, Page: p.Page, PageSize: p.PageSize})
}

func (a *App) handleCreateRole(c *gin.Context) {
	var req createRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" || req.Code == "" {
		fail(c, http.StatusBadRequest, 40020, "name and code required")
		return
	}
	dataScope := req.DataScope
	if dataScope == "" {
		dataScope = models.DataScopeSelf
	}
	role := models.Role{Name: req.Name, Code: req.Code, Description: req.Description, DataScope: dataScope}
	perms, err := a.loadPermissions(req.PermissionIDs)
	if err != nil {
		fail(c, http.StatusBadRequest, 40021, "invalid permission ids")
		return
	}
	if err := a.withTx(func(tx *gorm.DB) error {
		if err := tx.Create(&role).Error; err != nil {
			return err
		}
		return tx.Model(&role).Association("Permissions").Replace(perms)
	}); err != nil {
		if isUniqueViolation(err) {
			fail(c, http.StatusConflict, 40920, "role code already exists")
			return
		}
		fail(c, http.StatusInternalServerError, 50021, "failed to create role")
		return
	}
	if err := seed.SyncRolePolicies(a.Enforcer, role.Code, perms); err != nil {
		fail(c, http.StatusInternalServerError, 50022, "failed to sync rbac")
		return
	}
	role.Permissions = perms
	ok(c, toRoleDTO(role, true))
}

func (a *App) handleUpdateRole(c *gin.Context) {
	var role models.Role
	if err := a.DB.Preload("Permissions").First(&role, c.Param("id")).Error; err != nil {
		fail(c, http.StatusNotFound, 40420, "role not found")
		return
	}
	var req updateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, 40022, "invalid request body")
		return
	}
	if req.Name != nil && *req.Name != "" {
		role.Name = *req.Name
	}
	if req.Description != nil {
		role.Description = *req.Description
	}
	if req.DataScope != nil && *req.DataScope != "" {
		role.DataScope = *req.DataScope
	}
	if err := a.DB.Save(&role).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50023, "failed to update role")
		return
	}
	ok(c, toRoleDTO(role, true))
}

func (a *App) handleDeleteRole(c *gin.Context) {
	var role models.Role
	if err := a.DB.First(&role, c.Param("id")).Error; err != nil {
		fail(c, http.StatusNotFound, 40420, "role not found")
		return
	}
	if seed.IsSeedRole(role.Code) {
		fail(c, http.StatusBadRequest, 40023, "cannot delete seeded role")
		return
	}
	if err := seed.RemoveRole(a.Enforcer, role.Code); err != nil {
		fail(c, http.StatusInternalServerError, 50024, "failed to sync rbac")
		return
	}
	if err := a.DB.Select("Permissions", "Users").Delete(&role).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50025, "failed to delete role")
		return
	}
	ok(c, gin.H{"deleted": role.ID})
}

func (a *App) handleAssignRolePermissions(c *gin.Context) {
	var role models.Role
	if err := a.DB.First(&role, c.Param("id")).Error; err != nil {
		fail(c, http.StatusNotFound, 40420, "role not found")
		return
	}
	var req assignPermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, 40024, "invalid request body")
		return
	}
	perms, err := a.loadPermissions(req.PermissionIDs)
	if err != nil {
		fail(c, http.StatusBadRequest, 40021, "invalid permission ids")
		return
	}
	if err := a.DB.Model(&role).Association("Permissions").Replace(perms); err != nil {
		fail(c, http.StatusInternalServerError, 50021, "failed to assign permissions")
		return
	}
	if err := seed.SyncRolePolicies(a.Enforcer, role.Code, perms); err != nil {
		fail(c, http.StatusInternalServerError, 50022, "failed to sync rbac")
		return
	}
	role.Permissions = perms
	ok(c, toRoleDTO(role, true))
}

func (a *App) loadPermissions(ids []uint) ([]models.Permission, error) {
	ids = parseIDs(ids)
	if len(ids) == 0 {
		return []models.Permission{}, nil
	}
	var perms []models.Permission
	if err := a.DB.Where("id IN ?", ids).Find(&perms).Error; err != nil {
		return nil, err
	}
	if len(perms) != len(ids) {
		return nil, errString("permission not found")
	}
	return perms, nil
}
