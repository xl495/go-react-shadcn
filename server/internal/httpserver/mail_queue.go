package httpserver

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go-react-shadcn/internal/mailer"
	"go-react-shadcn/internal/models"
	"gorm.io/gorm"
)

type unsubscribeRequest struct {
	Token string `json:"token"`
}

type createCampaignRequest struct {
	Name     string `json:"name"`
	Subject  string `json:"subject"`
	Body     string `json:"body"`
	Audience string `json:"audience"`
}

type updateCampaignRequest struct {
	Name     *string `json:"name"`
	Subject  *string `json:"subject"`
	Body     *string `json:"body"`
	Audience *string `json:"audience"`
	Status   *string `json:"status"`
}

type scheduleCampaignRequest struct {
	ScheduledAt *time.Time `json:"scheduledAt"`
}

func parseIDParam(c *gin.Context) (uint, bool) {
	n, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || n == 0 {
		return 0, false
	}
	return uint(n), true
}

func (a *App) handleUnsubscribe(c *gin.Context) {
	var req unsubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, CodeInvalidBody, "invalid request body")
		return
	}
	id, okTok := mailer.ParseUnsubToken(a.Cfg.JWTSecret, req.Token)
	if !okTok {
		fail(c, http.StatusBadRequest, CodeInvalidUnsubToken, "invalid unsubscribe token")
		return
	}
	res := a.DB.Model(&models.User{}).Where("id = ?", id).Update("marketing_opt_in", false)
	if res.Error != nil {
		fail(c, http.StatusInternalServerError, CodeUpdateUser, "failed to update user")
		return
	}
	if res.RowsAffected == 0 {
		fail(c, http.StatusNotFound, CodeUserNotFound, "user not found")
		return
	}
	ok(c, gin.H{"unsubscribed": true})
}

func (a *App) handleListMailJobs(c *gin.Context) {
	p := parsePage(c, 20, 200)
	q := a.DB.Model(&models.MailJob{})
	q = applyEqual(q, "status", c.Query("status"))
	q = applyEqual(q, "class", c.Query("class"))
	var total int64
	_ = q.Count(&total).Error
	var rows []models.MailJob
	if err := q.Order("priority asc, id desc").Offset(p.Offset()).Limit(p.PageSize).Find(&rows).Error; err != nil {
		fail(c, http.StatusInternalServerError, CodeSendMail, "failed to list mail jobs")
		return
	}
	ok(c, pageResult[models.MailJob]{Items: rows, Total: total, Page: p.Page, PageSize: p.PageSize})
}

func (a *App) handleRetryMailJob(c *gin.Context) {
	id, okID := parseIDParam(c)
	if !okID {
		fail(c, http.StatusNotFound, CodeMailJobNotFound, "mail job not found")
		return
	}
	err := a.MailQ.Retry(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		fail(c, http.StatusNotFound, CodeMailJobNotFound, "mail job not found")
		return
	}
	if errors.Is(err, mailer.ErrCannotRetry) {
		fail(c, http.StatusBadRequest, CodeMailJobCannotRetry, "job cannot be retried")
		return
	}
	if err != nil {
		fail(c, http.StatusInternalServerError, CodeSendMail, "failed to retry mail job")
		return
	}
	a.MailQ.Poke()
	ok(c, gin.H{"retried": true, "id": id})
}

func (a *App) handleCancelMailJob(c *gin.Context) {
	id, okID := parseIDParam(c)
	if !okID {
		fail(c, http.StatusNotFound, CodeMailJobNotFound, "mail job not found")
		return
	}
	err := a.MailQ.Cancel(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		fail(c, http.StatusNotFound, CodeMailJobNotFound, "mail job not found")
		return
	}
	if err != nil {
		fail(c, http.StatusInternalServerError, CodeSendMail, "failed to cancel mail job")
		return
	}
	ok(c, gin.H{"canceled": true, "id": id})
}

func (a *App) handleListMailCampaigns(c *gin.Context) {
	p := parsePage(c, 20, 200)
	q := a.DB.Model(&models.MailCampaign{})
	q = applyEqual(q, "status", c.Query("status"))
	var total int64
	_ = q.Count(&total).Error
	var rows []models.MailCampaign
	if err := q.Order("id desc").Offset(p.Offset()).Limit(p.PageSize).Find(&rows).Error; err != nil {
		fail(c, http.StatusInternalServerError, CodeSendMail, "failed to list campaigns")
		return
	}
	ok(c, pageResult[models.MailCampaign]{Items: rows, Total: total, Page: p.Page, PageSize: p.PageSize})
}

func (a *App) handleGetMailCampaign(c *gin.Context) {
	id, okID := parseIDParam(c)
	if !okID {
		fail(c, http.StatusNotFound, CodeMailCampaignNotFound, "campaign not found")
		return
	}
	var row models.MailCampaign
	if err := a.DB.First(&row, id).Error; err != nil {
		fail(c, http.StatusNotFound, CodeMailCampaignNotFound, "campaign not found")
		return
	}
	ok(c, row)
}

func (a *App) handleCreateMailCampaign(c *gin.Context) {
	var req createCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Subject) == "" {
		fail(c, http.StatusBadRequest, CodeInvalidBody, "name and subject required")
		return
	}
	audience := strings.TrimSpace(req.Audience)
	if audience == "" {
		audience = models.AudienceOptedIn
	}
	if audience != models.AudienceOptedIn && audience != models.AudienceAllActive {
		fail(c, http.StatusBadRequest, CodeInvalidBody, "invalid audience")
		return
	}
	row := models.MailCampaign{
		Name:     strings.TrimSpace(req.Name),
		Subject:  strings.TrimSpace(req.Subject),
		Body:     req.Body,
		Audience: audience,
		Status:   models.CampaignDraft,
	}
	if err := a.DB.Create(&row).Error; err != nil {
		fail(c, http.StatusInternalServerError, CodeSendMail, "failed to create campaign")
		return
	}
	ok(c, row)
}

func (a *App) handleUpdateMailCampaign(c *gin.Context) {
	id, okID := parseIDParam(c)
	if !okID {
		fail(c, http.StatusNotFound, CodeMailCampaignNotFound, "campaign not found")
		return
	}
	var row models.MailCampaign
	if err := a.DB.First(&row, id).Error; err != nil {
		fail(c, http.StatusNotFound, CodeMailCampaignNotFound, "campaign not found")
		return
	}
	var req updateCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, CodeInvalidBody, "invalid request body")
		return
	}
	if req.Status != nil && *req.Status == models.CampaignPaused {
		if row.Status != models.CampaignScheduled && row.Status != models.CampaignRunning && row.Status != models.CampaignDraft {
			fail(c, http.StatusBadRequest, CodeMailCampaignState, "campaign cannot be paused")
			return
		}
		_ = a.MailQ.CancelCampaign(row.ID)
		row.Status = models.CampaignPaused
	} else if row.Status != models.CampaignDraft && row.Status != models.CampaignPaused {
		fail(c, http.StatusBadRequest, CodeMailCampaignState, "campaign cannot be edited")
		return
	}
	if req.Name != nil {
		row.Name = strings.TrimSpace(*req.Name)
	}
	if req.Subject != nil {
		row.Subject = strings.TrimSpace(*req.Subject)
	}
	if req.Body != nil {
		row.Body = *req.Body
	}
	if req.Audience != nil {
		audience := strings.TrimSpace(*req.Audience)
		if audience != models.AudienceOptedIn && audience != models.AudienceAllActive {
			fail(c, http.StatusBadRequest, CodeInvalidBody, "invalid audience")
			return
		}
		row.Audience = audience
	}
	if err := a.DB.Save(&row).Error; err != nil {
		fail(c, http.StatusInternalServerError, CodeSendMail, "failed to update campaign")
		return
	}
	ok(c, row)
}

func (a *App) handleDeleteMailCampaign(c *gin.Context) {
	id, okID := parseIDParam(c)
	if !okID {
		fail(c, http.StatusNotFound, CodeMailCampaignNotFound, "campaign not found")
		return
	}
	var row models.MailCampaign
	if err := a.DB.First(&row, id).Error; err != nil {
		fail(c, http.StatusNotFound, CodeMailCampaignNotFound, "campaign not found")
		return
	}
	if row.Status != models.CampaignDraft && row.Status != models.CampaignPaused {
		fail(c, http.StatusBadRequest, CodeMailCampaignState, "campaign cannot be deleted")
		return
	}
	_ = a.MailQ.CancelCampaign(row.ID)
	if err := a.DB.Delete(&row).Error; err != nil {
		fail(c, http.StatusInternalServerError, CodeSendMail, "failed to delete campaign")
		return
	}
	ok(c, gin.H{"deleted": row.ID})
}

func (a *App) handleScheduleMailCampaign(c *gin.Context) {
	id, okID := parseIDParam(c)
	if !okID {
		fail(c, http.StatusNotFound, CodeMailCampaignNotFound, "campaign not found")
		return
	}
	var row models.MailCampaign
	if err := a.DB.First(&row, id).Error; err != nil {
		fail(c, http.StatusNotFound, CodeMailCampaignNotFound, "campaign not found")
		return
	}
	if row.Status != models.CampaignDraft && row.Status != models.CampaignPaused {
		fail(c, http.StatusBadRequest, CodeMailCampaignState, "campaign cannot be scheduled")
		return
	}
	var req scheduleCampaignRequest
	_ = c.ShouldBindJSON(&req)
	when := time.Now()
	if req.ScheduledAt != nil && !req.ScheduledAt.IsZero() {
		when = *req.ScheduledAt
	}
	row.Status = models.CampaignScheduled
	row.ScheduledAt = &when
	if err := a.DB.Save(&row).Error; err != nil {
		fail(c, http.StatusInternalServerError, CodeSendMail, "failed to schedule campaign")
		return
	}
	a.MailQ.ProcessCampaigns(time.Now())
	a.MailQ.Poke()
	_ = a.DB.First(&row, row.ID)
	ok(c, row)
}
