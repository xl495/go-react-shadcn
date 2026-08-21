package httpserver

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/models"
)

func (a *App) notify(kind string, userID uint, typ, title, body, refType string, refID uint) {
	if userID == 0 {
		return
	}
	row := models.Notification{
		UserKind: models.NormalizeUserKind(kind),
		UserID:   userID,
		Type:     typ,
		Title:    title,
		Body:     body,
		RefType:  refType,
		RefID:    refID,
		CreatedAt: time.Now(),
	}
	_ = a.DB.Create(&row).Error
}

func (a *App) handleListNotifications(c *gin.Context) {
	claims := currentUser(c)
	if claims == nil {
		fail(c, http.StatusUnauthorized, CodeMissingToken, "missing bearer token")
		return
	}
	p := parsePage(c, 20, 100)
	kind := models.NormalizeUserKind(claimsKind(claims))
	q := a.DB.Model(&models.Notification{}).Where("user_kind = ? AND user_id = ?", kind, claims.UserID)
	if strings.EqualFold(c.Query("unread"), "1") || strings.EqualFold(c.Query("unread"), "true") {
		q = q.Where("read_at IS NULL")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		fail(c, http.StatusInternalServerError, CodeListNotifications, "failed to list notifications")
		return
	}
	var rows []models.Notification
	if err := q.Order("id desc").Offset(p.Offset()).Limit(p.PageSize).Find(&rows).Error; err != nil {
		fail(c, http.StatusInternalServerError, CodeListNotifications, "failed to list notifications")
		return
	}
	ok(c, pageResult[models.Notification]{Items: rows, Total: total, Page: p.Page, PageSize: p.PageSize})
}

func (a *App) handleUnreadNotificationCount(c *gin.Context) {
	claims := currentUser(c)
	if claims == nil {
		fail(c, http.StatusUnauthorized, CodeMissingToken, "missing bearer token")
		return
	}
	kind := models.NormalizeUserKind(claimsKind(claims))
	var n int64
	if err := a.DB.Model(&models.Notification{}).
		Where("user_kind = ? AND user_id = ? AND read_at IS NULL", kind, claims.UserID).
		Count(&n).Error; err != nil {
		fail(c, http.StatusInternalServerError, CodeListNotifications, "failed to list notifications")
		return
	}
	ok(c, gin.H{"unread": n})
}

func (a *App) handleReadNotification(c *gin.Context) {
	claims := currentUser(c)
	if claims == nil {
		fail(c, http.StatusUnauthorized, CodeMissingToken, "missing bearer token")
		return
	}
	kind := models.NormalizeUserKind(claimsKind(claims))
	var row models.Notification
	if err := a.DB.Where("id = ? AND user_kind = ? AND user_id = ?", c.Param("id"), kind, claims.UserID).First(&row).Error; err != nil {
		fail(c, http.StatusNotFound, CodeNotificationNotFound, "notification not found")
		return
	}
	now := time.Now()
	if row.ReadAt == nil {
		_ = a.DB.Model(&row).Update("read_at", now).Error
		row.ReadAt = &now
	}
	ok(c, row)
}

func (a *App) handleReadAllNotifications(c *gin.Context) {
	claims := currentUser(c)
	if claims == nil {
		fail(c, http.StatusUnauthorized, CodeMissingToken, "missing bearer token")
		return
	}
	kind := models.NormalizeUserKind(claimsKind(claims))
	now := time.Now()
	res := a.DB.Model(&models.Notification{}).
		Where("user_kind = ? AND user_id = ? AND read_at IS NULL", kind, claims.UserID).
		Update("read_at", now)
	if res.Error != nil {
		fail(c, http.StatusInternalServerError, CodeListNotifications, "failed to list notifications")
		return
	}
	ok(c, gin.H{"updated": res.RowsAffected})
}

type announceRequest struct {
	Kind  string `json:"kind"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

func (a *App) handleCreateAnnouncement(c *gin.Context) {
	var req announceRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Title) == "" {
		fail(c, http.StatusBadRequest, CodeInvalidBody, "title required")
		return
	}
	kind := models.NormalizeUserKind(req.Kind)
	if kind != models.UserKindAdmin && kind != models.UserKindWeb {
		fail(c, http.StatusBadRequest, CodeInvalidUserBody, "invalid user kind")
		return
	}
	var ids []uint
	if err := a.accounts(kind).Select("id").Pluck("id", &ids).Error; err != nil {
		fail(c, http.StatusInternalServerError, CodeListNotifications, "failed to list notifications")
		return
	}
	now := time.Now()
	rows := make([]models.Notification, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, models.Notification{
			UserKind: kind, UserID: id, Type: "announce",
			Title: strings.TrimSpace(req.Title), Body: strings.TrimSpace(req.Body),
			RefType: "announce", CreatedAt: now,
		})
	}
	if len(rows) > 0 {
		if err := a.DB.Create(&rows).Error; err != nil {
			fail(c, http.StatusInternalServerError, CodeListNotifications, "failed to list notifications")
			return
		}
	}
	ok(c, gin.H{"sent": len(rows)})
}
