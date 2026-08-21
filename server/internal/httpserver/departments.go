package httpserver

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/seed"
	"gorm.io/gorm"
)

func (a *App) withTx(fn func(tx *gorm.DB) error) error {
	return a.DB.Transaction(fn)
}

func (a *App) loadDepartments() ([]models.Department, error) {
	if rows, ok := a.depts.get(); ok {
		return rows, nil
	}
	var rows []models.Department
	if err := a.DB.Order("sort asc, id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	a.depts.put(rows)
	return rows, nil
}

func (a *App) refreshDepartments() {
	_ = seed.SyncDepartmentDict(a.DB)
	a.depts.invalidate()
}

func (a *App) resolveDepartmentID(code string) *uint {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil
	}
	rows, err := a.loadDepartments()
	if err != nil {
		return nil
	}
	for _, d := range rows {
		if d.Code == code {
			id := d.ID
			return &id
		}
	}
	return nil
}

func (a *App) departmentCodeOK(code string) bool {
	code = strings.TrimSpace(code)
	if code == "" {
		return true
	}
	rows, err := a.loadDepartments()
	if err != nil {
		return false
	}
	for _, d := range rows {
		if d.Code == code && d.Status != "disabled" {
			return true
		}
	}
	return false
}

func (a *App) requireDepartmentCode(c *gin.Context, code string) bool {
	if a.departmentCodeOK(code) {
		return true
	}
	fail(c, http.StatusBadRequest, CodeInvalidDept, "invalid department")
	return false
}

func (a *App) applyDepartmentFilter(q *gorm.DB, tbl, code string) *gorm.DB {
	code = strings.TrimSpace(code)
	if code == "" {
		return q
	}
	id := a.resolveDepartmentID(code)
	if id == nil {
		return q.Where("1 = 0")
	}
	return q.Where(tbl+".department_id = ?", *id)
}

func (a *App) applyDepartmentLink(user *models.User) {
	code := strings.TrimSpace(user.Department)
	if code != "" {
		user.DepartmentID = a.resolveDepartmentID(code)
		if user.DepartmentID == nil {
			user.Department = ""
			return
		}
		user.Department = code
		return
	}
	a.fillUserDepartments(user)
}

func (a *App) fillUserDepartments(users ...*models.User) {
	if len(users) == 0 {
		return
	}
	rows, err := a.loadDepartments()
	if err != nil {
		return
	}
	byID := make(map[uint]string, len(rows))
	for _, d := range rows {
		byID[d.ID] = d.Code
	}
	for _, u := range users {
		if u == nil {
			continue
		}
		if u.DepartmentID == nil {
			u.Department = ""
			continue
		}
		code, ok := byID[*u.DepartmentID]
		if !ok {
			u.Department = ""
			u.DepartmentID = nil
			continue
		}
		u.Department = code
	}
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "unique")
}

func (a *App) handleListDepartments(c *gin.Context) {
	rows, err := a.loadDepartments()
	if err != nil {
		fail(c, http.StatusInternalServerError, CodeListDepts, "failed to list departments")
		return
	}
	rows = filterDepartments(rows, c.Query("q"))
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

func filterDepartments(rows []models.Department, kw string) []models.Department {
	kw = strings.ToLower(strings.TrimSpace(kw))
	if kw == "" {
		return rows
	}
	byID := make(map[uint]models.Department, len(rows))
	matched := make(map[uint]bool)
	for _, d := range rows {
		byID[d.ID] = d
		if strings.Contains(strings.ToLower(d.Name), kw) ||
			strings.Contains(strings.ToLower(d.Code), kw) ||
			strings.Contains(strings.ToLower(d.Leader), kw) {
			matched[d.ID] = true
		}
	}
	keep := make(map[uint]bool, len(matched))
	for id := range matched {
		for cur := id; !keep[cur]; {
			keep[cur] = true
			p := byID[cur].ParentID
			if p == nil {
				break
			}
			cur = *p
		}
	}
	out := make([]models.Department, 0, len(keep))
	for _, d := range rows {
		if keep[d.ID] {
			out = append(out, d)
		}
	}
	return out
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
	if a.rejectBadDepartmentParent(c, 0, req.ParentID) {
		return
	}
	row := models.Department{
		Name: req.Name, Code: req.Code, ParentID: req.ParentID,
		Sort: req.Sort, Leader: req.Leader, Status: req.Status,
	}
	if err := a.DB.Create(&row).Error; err != nil {
		fail(c, http.StatusConflict, 40990, "department code already exists")
		return
	}
	a.refreshDepartments()
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
	if a.rejectBadDepartmentParent(c, row.ID, row.ParentID) {
		return
	}
	if err := a.DB.Save(&row).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50091, "failed to update department")
		return
	}
	_ = models.Accounts(a.DB, models.UserKindAdmin).Where("department_id = ?", row.ID).Update("department", row.Code).Error
	_ = models.Accounts(a.DB, models.UserKindWeb).Where("department_id = ?", row.ID).Update("department", row.Code).Error
	a.refreshDepartments()
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
	if a.departmentUserCount(row.ID) > 0 {
		fail(c, http.StatusBadRequest, CodeDeptHasUsers, "department still has users")
		return
	}
	if err := a.DB.Delete(&row).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50092, "failed to delete department")
		return
	}
	a.refreshDepartments()
	ok(c, gin.H{"deleted": row.ID})
}

func (a *App) departmentUserCount(id uint) int64 {
	var adminN, webN int64
	_ = models.Accounts(a.DB, models.UserKindAdmin).Where("department_id = ?", id).Count(&adminN).Error
	_ = models.Accounts(a.DB, models.UserKindWeb).Where("department_id = ?", id).Count(&webN).Error
	return adminN + webN
}

func (a *App) rejectBadDepartmentParent(c *gin.Context, id uint, parentID *uint) bool {
	if parentID == nil {
		return false
	}
	if id != 0 && *parentID == id {
		fail(c, http.StatusBadRequest, CodeDeptCycle, "department cannot be its own parent")
		return true
	}
	var parent models.Department
	if err := a.DB.Select("id").First(&parent, *parentID).Error; err != nil {
		fail(c, http.StatusNotFound, CodeDeptNotFound, "parent department not found")
		return true
	}
	if id != 0 && a.deptWouldCycle(id, *parentID) {
		fail(c, http.StatusBadRequest, CodeDeptCycle, "department parent would create a cycle")
		return true
	}
	return false
}

func (a *App) deptWouldCycle(id, parentID uint) bool {
	seen := map[uint]struct{}{id: {}}
	cur := parentID
	for range 64 {
		if _, ok := seen[cur]; ok {
			return true
		}
		seen[cur] = struct{}{}
		var row models.Department
		if err := a.DB.Select("id", "parent_id").First(&row, cur).Error; err != nil {
			return false
		}
		if row.ParentID == nil {
			return false
		}
		cur = *row.ParentID
	}
	return true
}
