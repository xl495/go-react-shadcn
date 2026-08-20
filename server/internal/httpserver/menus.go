package httpserver

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/models"
	"gorm.io/gorm"
)

type menuNode struct {
	ID          uint       `json:"id"`
	Name        string     `json:"name"`
	Code        string     `json:"code"`
	Kind        string     `json:"kind"`
	RoutePath   string     `json:"routePath"`
	Component   string     `json:"component"`
	Icon        string     `json:"icon"`
	Sort        int        `json:"sort"`
	Hidden      bool       `json:"hidden"`
	Children    []menuNode `json:"children,omitempty"`
}

func (a *App) handleMenus(c *gin.Context) {
	claims := currentUser(c)
	var user models.User
	if err := a.DB.Preload("Roles.Permissions").First(&user, claims.UserID).Error; err != nil {
		fail(c, http.StatusNotFound, 40401, "user not found")
		return
	}
	codes := collectCodes(user)
	isAdmin := hasRole(user.Roles, "admin") || containsCode(codes, "admin:all")

	var perms []models.Permission
	if err := a.DB.Order("sort asc, id asc").Find(&perms).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50093, "failed to list menus")
		return
	}
	filtered := make([]models.Permission, 0, len(perms))
	for _, p := range perms {
		if p.Kind != "menu" && p.Kind != "button" {
			continue
		}
		if isAdmin || containsCode(codes, p.Code) {
			filtered = append(filtered, p)
		}
	}
	ok(c, buildMenuTree(filtered, nil))
}

func buildMenuTree(perms []models.Permission, parentID *uint) []menuNode {
	out := make([]menuNode, 0)
	for _, p := range perms {
		same := (parentID == nil && p.ParentID == nil) ||
			(parentID != nil && p.ParentID != nil && *parentID == *p.ParentID)
		if !same {
			continue
		}
		out = append(out, menuNode{
			ID: p.ID, Name: p.Name, Code: p.Code, Kind: p.Kind,
			RoutePath: p.RoutePath, Component: p.Component, Icon: p.Icon,
			Sort: p.Sort, Hidden: p.Hidden,
			Children: buildMenuTree(perms, &p.ID),
		})
	}
	return out
}

func containsCode(codes []string, code string) bool {
	for _, c := range codes {
		if c == code || c == "*" {
			return true
		}
	}
	return false
}

func (a *App) userDataScope(user models.User) string {
	scope := models.DataScopeSelf
	for _, r := range user.Roles {
		switch r.DataScope {
		case models.DataScopeAll:
			return models.DataScopeAll
		case models.DataScopeDeptAndSub:
			if scope != models.DataScopeAll {
				scope = models.DataScopeDeptAndSub
			}
		case models.DataScopeDept:
			if scope == models.DataScopeSelf {
				scope = models.DataScopeDept
			}
		}
	}
	return scope
}

func (a *App) applyUserDataScope(q *gorm.DB, user models.User) *gorm.DB {
	scope := a.userDataScope(user)
	switch scope {
	case models.DataScopeAll:
		return q
	case models.DataScopeSelf:
		return q.Where("id = ?", user.ID)
	case models.DataScopeDept:
		if user.DepartmentID == nil {
			return q.Where("id = ?", user.ID)
		}
		return q.Where("department_id = ?", *user.DepartmentID)
	case models.DataScopeDeptAndSub:
		if user.DepartmentID == nil {
			return q.Where("id = ?", user.ID)
		}
		ids := a.deptSubtreeIDs(*user.DepartmentID)
		if len(ids) == 0 {
			return q.Where("department_id = ?", *user.DepartmentID)
		}
		return q.Where("department_id IN ?", ids)
	default:
		return q.Where("id = ?", user.ID)
	}
}

func (a *App) deptSubtreeIDs(root uint) []uint {
	var rows []models.Department
	_ = a.DB.Find(&rows).Error
	out := []uint{root}
	var walk func(id uint)
	walk = func(id uint) {
		for _, d := range rows {
			if d.ParentID != nil && *d.ParentID == id {
				out = append(out, d.ID)
				walk(d.ID)
			}
		}
	}
	walk(root)
	return out
}

func isAnomalousLogin(user models.User, ip string) bool {
	if user.LastLoginIP == "" || ip == "" {
		return false
	}
	if user.LastLoginIP == ip {
		return false
	}
	// Simple heuristic: different /16 prefix treated as anomalous.
	partsA := strings.Split(user.LastLoginIP, ".")
	partsB := strings.Split(ip, ".")
	if len(partsA) >= 2 && len(partsB) >= 2 {
		return partsA[0] != partsB[0] || partsA[1] != partsB[1]
	}
	return user.LastLoginIP != ip
}
