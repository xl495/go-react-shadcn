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
	if err := q.DB.Where("status = ? AND scheduled_at <= ?", models.CampaignScheduled, now).Find(&due).Error; err != nil {
		return
	}
	for i := range due {
		q.fanOut(&due[i], now)
	}
	q.finishCampaigns()
}

func (q *Queue) fanOut(c *models.MailCampaign, now time.Time) {
	query := q.DB.Where("status = ? AND email <> ''", "active")
	if c.Audience != models.AudienceAllActive {
		query = query.Where("marketing_opt_in = ?", true)
	}
	var users []models.User
	if err := query.Find(&users).Error; err != nil {
		return
	}
	started := now
	_ = q.DB.Model(c).Updates(map[string]any{
		"status":     models.CampaignRunning,
		"started_at": started,
	}).Error
	cfg, _ := Load(q.DB)
	n := 0
	for i := range users {
		u := users[i]
		unsub := ""
		if q.Secret != "" {
			unsub = UnsubLink(cfg.ResetBaseURL, q.Secret, u.ID)
		}
		subject, body := RenderMailTemplate(c.Subject, c.Body, &u, unsub)
		key := "campaign:" + strconv.FormatUint(uint64(c.ID), 10) + ":user:" + strconv.FormatUint(uint64(u.ID), 10)
		if _, err := q.Enqueue(EnqueueInput{
			Class:      models.MailClassMarketing,
			User:       &u,
			Subject:    subject,
			Body:       body,
			CampaignID: &c.ID,
			DedupeKey:  key,
			Now:        now,
		}); err == nil {
			n++
		}
	}
	updates := map[string]any{"job_count": n}
	if n == 0 {
		updates["status"] = models.CampaignDone
		updates["finished_at"] = now
	}
	_ = q.DB.Model(c).Updates(updates).Error
}

func (q *Queue) finishCampaigns() {
	var running []models.MailCampaign
	if err := q.DB.Where("status = ?", models.CampaignRunning).Find(&running).Error; err != nil {
		return
	}
	for i := range running {
		c := running[i]
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
