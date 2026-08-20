package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/models"
	"gorm.io/gorm"
)

type dictTypeRequest struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Remark string `json:"remark"`
}

type dictItemRequest struct {
	Label  string `json:"label"`
	Value  string `json:"value"`
	Sort   int    `json:"sort"`
	Status string `json:"status"`
	Remark string `json:"remark"`
}

func (a *App) handleListDicts(c *gin.Context) {
	p := parsePage(c, 50, 500)
	q := a.DB.Model(&models.DictType{})
	var total int64
	_ = q.Count(&total).Error
	var rows []models.DictType
	if err := q.Order("id asc").Offset(p.Offset()).Limit(p.PageSize).Find(&rows).Error; err != nil {
		fail(c, http.StatusInternalServerError, CodeListDicts, "failed to list dicts")
		return
	}
	ok(c, pageResult[models.DictType]{Items: rows, Total: total, Page: p.Page, PageSize: p.PageSize})
}

func (a *App) handleCreateDict(c *gin.Context) {
	var req dictTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Code == "" || req.Name == "" {
		fail(c, http.StatusBadRequest, 40060, "code and name required")
		return
	}
	if req.Status == "" {
		req.Status = "active"
	}
	row := models.DictType{Code: req.Code, Name: req.Name, Status: req.Status, Remark: req.Remark}
	if err := a.DB.Create(&row).Error; err != nil {
		fail(c, http.StatusConflict, 40960, "dict code already exists")
		return
	}
	ok(c, row)
}

func (a *App) handleUpdateDict(c *gin.Context) {
	var row models.DictType
	if err := a.DB.First(&row, c.Param("id")).Error; err != nil {
		fail(c, http.StatusNotFound, 40460, "dict not found")
		return
	}
	var req dictTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, 40061, "invalid request body")
		return
	}
	if req.Name != "" {
		row.Name = req.Name
	}
	if req.Status != "" {
		row.Status = req.Status
	}
	row.Remark = req.Remark
	if err := a.DB.Save(&row).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50061, "failed to update dict")
		return
	}
	ok(c, row)
}

func (a *App) handleDeleteDict(c *gin.Context) {
	var row models.DictType
	if err := a.DB.First(&row, c.Param("id")).Error; err != nil {
		fail(c, http.StatusNotFound, 40460, "dict not found")
		return
	}
	if err := a.withTx(func(tx *gorm.DB) error {
		if err := tx.Where("type_code = ?", row.Code).Delete(&models.DictItem{}).Error; err != nil {
			return err
		}
		return tx.Delete(&row).Error
	}); err != nil {
		fail(c, http.StatusInternalServerError, CodeDeleteDict, "failed to delete dict")
		return
	}
	ok(c, gin.H{"deleted": row.ID})
}

func (a *App) handleListDictItems(c *gin.Context) {
	var typ models.DictType
	if err := a.DB.First(&typ, c.Param("id")).Error; err != nil {
		fail(c, http.StatusNotFound, 40460, "dict not found")
		return
	}
	var items []models.DictItem
	p := parsePage(c, 100, 500)
	q := a.DB.Model(&models.DictItem{}).Where("type_code = ?", typ.Code)
	var total int64
	_ = q.Count(&total).Error
	if err := q.Order("sort asc, id asc").Offset(p.Offset()).Limit(p.PageSize).Find(&items).Error; err != nil {
		fail(c, http.StatusInternalServerError, CodeListDictItems, "failed to list dict items")
		return
	}
	ok(c, pageResult[models.DictItem]{Items: items, Total: total, Page: p.Page, PageSize: p.PageSize})
}

func (a *App) handleCreateDictItem(c *gin.Context) {
	var typ models.DictType
	if err := a.DB.First(&typ, c.Param("id")).Error; err != nil {
		fail(c, http.StatusNotFound, 40460, "dict not found")
		return
	}
	var req dictItemRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Label == "" || req.Value == "" {
		fail(c, http.StatusBadRequest, 40062, "label and value required")
		return
	}
	if req.Status == "" {
		req.Status = "active"
	}
	row := models.DictItem{TypeCode: typ.Code, Label: req.Label, Value: req.Value, Sort: req.Sort, Status: req.Status, Remark: req.Remark}
	if err := a.DB.Create(&row).Error; err != nil {
		fail(c, http.StatusConflict, 40961, "dict item already exists")
		return
	}
	ok(c, row)
}

func (a *App) handleUpdateDictItem(c *gin.Context) {
	var row models.DictItem
	if err := a.DB.First(&row, c.Param("id")).Error; err != nil {
		fail(c, http.StatusNotFound, 40461, "dict item not found")
		return
	}
	var req dictItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, 40063, "invalid request body")
		return
	}
	if req.Label != "" {
		row.Label = req.Label
	}
	if req.Value != "" {
		row.Value = req.Value
	}
	row.Sort = req.Sort
	if req.Status != "" {
		row.Status = req.Status
	}
	row.Remark = req.Remark
	if err := a.DB.Save(&row).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50065, "failed to update dict item")
		return
	}
	ok(c, row)
}

func (a *App) handleDeleteDictItem(c *gin.Context) {
	var row models.DictItem
	if err := a.DB.First(&row, c.Param("id")).Error; err != nil {
		fail(c, http.StatusNotFound, 40461, "dict item not found")
		return
	}
	if err := a.DB.Delete(&row).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50066, "failed to delete dict item")
		return
	}
	ok(c, gin.H{"deleted": row.ID})
}

func (a *App) handleLookupDict(c *gin.Context) {
	code := c.Param("code")
	var typ models.DictType
	if err := a.DB.Where("code = ? AND status = ?", code, "active").First(&typ).Error; err != nil {
		fail(c, http.StatusNotFound, 40460, "dict not found")
		return
	}
	var items []models.DictItem
	if err := a.DB.Where("type_code = ? AND status = ?", code, "active").Order("sort asc, id asc").Find(&items).Error; err != nil {
		fail(c, http.StatusInternalServerError, 50064, "failed to list dict items")
		return
	}
	ok(c, gin.H{"code": typ.Code, "name": typ.Name, "items": items})
}

func (a *App) dictValueOK(code, value string) bool {
	if value == "" {
		return true
	}
	var n int64
	if err := a.DB.Model(&models.DictItem{}).
		Where("type_code = ? AND value = ? AND status = ?", code, value, "active").
		Count(&n).Error; err != nil {
		return false
	}
	return n > 0
}

func (a *App) requireDictValue(c *gin.Context, code, value string) bool {
	if a.dictValueOK(code, value) {
		return true
	}
	fail(c, http.StatusBadRequest, 40015, "invalid dictionary value")
	return false
}
