package mailer

import (
	"strconv"
	"time"

	"go-react-shadcn/internal/models"
)

func (q *Queue) ProcessCampaigns(now time.Time) {
	if q == nil || q.DB == nil {
		return
	}
	var due []models.MailCampaign
	if err := q.DB.Where(
		"(status = ? AND scheduled_at <= ?) OR status = ?",
		models.CampaignScheduled, now, models.CampaignRunning,
	).Find(&due).Error; err != nil {
		return
	}
	for i := range due {
		q.fanOut(&due[i], now)
	}
	q.finishCampaigns()
}

const campaignFanoutBatch = 300

func (q *Queue) fanOut(c *models.MailCampaign, now time.Time) {
	if c.Status != models.CampaignRunning {
		started := now
		_ = q.DB.Model(c).Updates(map[string]any{
			"status":     models.CampaignRunning,
			"started_at": started,
		}).Error
		c.Status = models.CampaignRunning
	}
	cfg, _ := Load(q.DB, q.ConfigKey)
	lastID := q.campaignLastUserID(c.ID)
	for {
		query := models.Accounts(q.DB, models.UserKindWeb).
			Select("id", "username", "nickname", "email", "timezone").
			Where("status = ? AND email <> '' AND id > ?", "active", lastID)
		if c.Audience != models.AudienceAllActive {
			query = query.Where("marketing_opt_in = ?", true)
		}
		var batch []models.User
		if err := query.Order("id asc").Limit(campaignFanoutBatch).Find(&batch).Error; err != nil {
			return
		}
		if len(batch) == 0 {
			break
		}
		for i := range batch {
			u := batch[i]
			u.Kind = models.UserKindWeb
			unsub := ""
			if q.Secret != "" {
				unsub = UnsubLink(cfg.ResetBaseURL, q.Secret, u.Kind, u.ID)
			}
			subject, body := RenderMailTemplate(c.Subject, c.Body, &u, unsub)
			key := "campaign:" + strconv.FormatUint(uint64(c.ID), 10) + ":user:" + strconv.FormatUint(uint64(u.ID), 10)
			_, _ = q.Enqueue(EnqueueInput{
				Class:      models.MailClassMarketing,
				User:       &u,
				Subject:    subject,
				Body:       body,
				CampaignID: &c.ID,
				DedupeKey:  key,
				Now:        now,
			})
		}
		lastID = batch[len(batch)-1].ID
	}
	var total int64
	_ = q.DB.Model(&models.MailJob{}).Where("campaign_id = ?", c.ID).Count(&total).Error
	updates := map[string]any{"job_count": int(total)}
	if total == 0 {
		updates["status"] = models.CampaignDone
		updates["finished_at"] = now
	}
	_ = q.DB.Model(c).Updates(updates).Error
}

func (q *Queue) campaignLastUserID(campaignID uint) uint {
	var last *uint
	_ = q.DB.Model(&models.MailJob{}).
		Where("campaign_id = ? AND user_id IS NOT NULL", campaignID).
		Select("MAX(user_id)").
		Scan(&last).Error
	if last == nil {
		return 0
	}
	return *last
}

func (q *Queue) finishCampaigns() {
	var running []models.MailCampaign
	if err := q.DB.Where("status = ?", models.CampaignRunning).Find(&running).Error; err != nil {
		return
	}
	for i := range running {
		c := running[i]
		if q.campaignHasPendingAudience(c) {
			continue
		}
		var open int64
		_ = q.DB.Model(&models.MailJob{}).
			Where("campaign_id = ? AND status IN ?", c.ID, []string{models.MailStatusQueued, models.MailStatusSending}).
			Count(&open).Error
		if open > 0 {
			continue
		}
		now := time.Now()
		_ = q.DB.Model(&c).Updates(map[string]any{
			"status":      models.CampaignDone,
			"finished_at": now,
		}).Error
	}
}

func (q *Queue) campaignHasPendingAudience(c models.MailCampaign) bool {
	lastID := q.campaignLastUserID(c.ID)
	query := models.Accounts(q.DB, models.UserKindWeb).
		Select("id").
		Where("status = ? AND email <> '' AND id > ?", "active", lastID)
	if c.Audience != models.AudienceAllActive {
		query = query.Where("marketing_opt_in = ?", true)
	}
	var id uint
	if err := query.Order("id asc").Limit(1).Scan(&id).Error; err != nil {
		return false
	}
	return id > 0
}
