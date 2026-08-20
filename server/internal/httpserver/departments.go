package httpserver

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/models"
	"gorm.io/gorm"
)

func (a *App) withTx(fn func(tx *gorm.DB) error) error {
	return a.DB.Transaction(fn)
}

func (a *App) resolveDepartmentID(code string) *uint {
	if code == "" {
		return nil
	}
	var dept models.Department
	if err := a.DB.Where("code = ?", code).First(&dept).Error; err != nil {
		return nil
	}
	return &dept.ID
}

func (a *App) applyDepartmentLink(user *models.User) {
	user.DepartmentID = a.resolveDepartmentID(user.Department)
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "unique")
}

func (a *App) handleListDepartments(c *gin.Context) {
	var rows []models.Department
	if err := a.DB.Order("sort asc, id asc").Find(&rows).Error; err != nil {
		fail(c, http.StatusInternalServerError, CodeListDepts, "failed to list departments")
		return
	}
	tree := buildDeptTree(rows, nil)
	p := parsePage(c, 50, 500)
	total := int64(len(tree))
	start := p.Offset()
	if start > len(tree) {
		start = len(tree)
	}
	end := start + p.PageSize
	if end > len(tree) {
		end = len(tree)
	}
	ok(c, pageResult[models.Department]{Items: tree[start:end], Total: total, Page: p.Page, PageSize: p.PageSize})
}

func buildDeptTree(rows []models.Department, parentID *uint) []models.Department {
	out := make([]models.Department, 0)
	for _, d := range rows {
		same := (parentID == nil && d.ParentID == nil) ||
			(parentID != nil && d.ParentID != nil && *parentID == *d.ParentID)
		if !same {
			continue
		}
		d.Children = buildDeptTree(rows, &d.ID)
		out = append(out, d)
	}
	return out
}

type deptRequest struct {
	Name     string `json:"name"`
	Code     string `json:"code"`
	ParentID *uint  `json:"parentId"`
	Sort     int    `json:"sort"`
	Leader   string `json:"leader"`
	Status   string `json:"status"`
}

func (a *App) handleCreateDepartment(c *gin.Context) {
	var req deptRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" || req.Code == "" {
		fail(c, http.StatusBadRequest, 40090, "name and code required")
		return
	}
	if req.Status == "" {
		req.Status = "active"
	}
	row := models.Department{
		Name: req.Name, Code: req.Code, ParentID: req.ParentID,
		Sort: req.Sort, Leader: req.Leader, Status: req.Status,
	}
	if err := a.DB.Create(&row).Error; err != nil {
		fail(c, http.StatusConflict, 40990, "department code already exists")
		return
	}
	ok(c, row)
}

func (a *App) handleUpdateDepartment(c *gin.Context) {
	var row models.Department
	if err := a.DB.First(&row, c.Param("id")).Error; err != nil {
		fail(c, http.StatusNotFound, 40490, "department not found")
		return
	}
	var req deptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, 40091, "invalid request body")
		return
	}
	if req.Name != "" {
		row.Name = req.Name
	}
	if req.Code != "" {
		row.Code = req.Code
	}
	row.ParentID = req.ParentID
	row.Sort = req.Sort
	row.Leader = req.Leader
	if req.Status != "" {
		row.Status = req.Status
	}
	if err := a.DB.Save(&row).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50091, "failed to update department")
		return
	}
	ok(c, row)
}

func (a *App) handleDeleteDepartment(c *gin.Context) {
	var row models.Department
	if err := a.DB.First(&row, c.Param("id")).Error; err != nil {
		fail(c, http.StatusNotFound, 40490, "department not found")
		return
	}
	var childCount int64
	a.DB.Model(&models.Department{}).Where("parent_id = ?", row.ID).Count(&childCount)
	if childCount > 0 {
		fail(c, http.StatusBadRequest, 40092, "department has children")
		return
	}
	if err := a.DB.Delete(&row).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50092, "failed to delete department")
		return
	}
	ok(c, gin.H{"deleted": row.ID})
}
