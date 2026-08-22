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
	q = applyContains(q, c.Query("q"), "name", "code", "description")
	total, okCount := countOrFail(c, q, CodeListRoles, "failed to list roles")
	if !okCount {
		return
	}
	var roles []models.Role
	if err := q.Order("id asc").Offset(p.Offset()).Limit(p.PageSize).Find(&roles).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50020, "failed to list roles")
		return
	}
	ids := make([]uint, 0, len(roles))
	for _, r := range roles {
		ids = append(ids, r.ID)
	}
	permIDs, err := a.rolePermissionIDs(ids)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50020, "failed to list roles")
		return
	}
	out := make([]roleDTO, 0, len(roles))
	for _, r := range roles {
		dto := toRoleDTO(r, false)
		dto.PermissionIDs = permIDs[r.ID]
		out = append(out, dto)
	}
	ok(c, pageResult[roleDTO]{Items: out, Total: total, Page: p.Page, PageSize: p.PageSize})
}

func (a *App) handleGetRole(c *gin.Context) {
	var role models.Role
	if err := a.DB.First(&role, c.Param("id")).Error; err != nil {
		fail(c, http.StatusNotFound, 40420, "role not found")
		return
	}
	ids, err := a.rolePermissionIDs([]uint{role.ID})
	if err != nil {
		fail(c, http.StatusInternalServerError, 50020, "failed to get role")
		return
	}
	dto := toRoleDTO(role, false)
	dto.PermissionIDs = ids[role.ID]
	ok(c, dto)
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
	if !validDataScope(dataScope) {
		fail(c, http.StatusBadRequest, CodeInvalidDataScope, "invalid data scope")
		return
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
		fail(c, http.StatusInternalServerError, CodeCreateRole, "failed to create role")
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
		if !validDataScope(*req.DataScope) {
			fail(c, http.StatusBadRequest, CodeInvalidDataScope, "invalid data scope")
			return
		}
		role.DataScope = *req.DataScope
	}
	if err := a.DB.Save(&role).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50023, "failed to update role")
		return
	}
	ok(c, toRoleDTO(role, true))
}

func validDataScope(v string) bool {
	switch v {
	case models.DataScopeAll, models.DataScopeDept, models.DataScopeDeptAndSub, models.DataScopeSelf:
		return true
	}
	return false
}

func (a *App) handleDeleteRole(c *gin.Context) {
	var role models.Role
	if err := a.DB.First(&role, c.Param("id")).Error; err != nil {
		fail(c, http.StatusNotFound, 40420, "role not found")
		return
	}
	if seed.IsSeedRole(role.Code) {
		fail(c, http.StatusBadRequest, CodeCannotDeleteRole, "cannot delete seeded role")
		return
	}
	if err := a.withTx(func(tx *gorm.DB) error {
		return tx.Select("Permissions", "Users").Delete(&role).Error
	}); err != nil {
		fail(c, http.StatusInternalServerError, CodeDeleteRole, "failed to delete role")
		return
	}
	if err := seed.RemoveRole(a.Enforcer, role.Code); err != nil {
		fail(c, http.StatusInternalServerError, CodeSyncRBAC, "failed to sync rbac")
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
	if err := a.withTx(func(tx *gorm.DB) error {
		return tx.Model(&role).Association("Permissions").Replace(perms)
	}); err != nil {
		fail(c, http.StatusInternalServerError, CodeAssignPerms, "failed to assign permissions")
		return
	}
	if err := seed.SyncRolePolicies(a.Enforcer, role.Code, perms); err != nil {
		fail(c, http.StatusInternalServerError, 50022, "failed to sync rbac")
		return
	}
	role.Permissions = perms
	a.notifyRoleUsers(role.ID, "perms", "角色权限已更新", role.Name)
	ok(c, toRoleDTO(role, true))
}

func (a *App) notifyRoleUsers(roleID uint, typ, title, body string) {
	var adminIDs []uint
	_ = a.DB.Table("user_roles").Where("role_id = ?", roleID).Pluck("user_id", &adminIDs).Error
	for _, id := range adminIDs {
		a.notify(models.UserKindAdmin, id, typ, title, body, "role", roleID)
	}
	var webIDs []uint
	_ = a.DB.Table("web_user_roles").Where("role_id = ?", roleID).Pluck("user_id", &webIDs).Error
	for _, id := range webIDs {
		a.notify(models.UserKindWeb, id, typ, title, body, "role", roleID)
	}
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

func (a *App) rolePermissionIDs(roleIDs []uint) (map[uint][]uint, error) {
	out := map[uint][]uint{}
	if len(roleIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		RoleID       uint `gorm:"column:role_id"`
		PermissionID uint `gorm:"column:permission_id"`
	}
	if err := a.DB.Table("role_permissions").Select("role_id, permission_id").Where("role_id IN ?", roleIDs).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.RoleID] = append(out[row.RoleID], row.PermissionID)
	}
	return out, nil
}
