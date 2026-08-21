package mailer

import (
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go-react-shadcn/internal/models"
	"gorm.io/gorm"
)

var ErrCannotRetry = errors.New("job cannot be retried")

type Queue struct {
	DB        *gorm.DB
	sender    Sender
	Secret    string
	ConfigKey string
	poke      chan struct{}
	stop      chan struct{}
	done      chan struct{}
	running   atomic.Bool
	mu        sync.Mutex
}

func NewQueue(db *gorm.DB, sender Sender, secret string) *Queue {
	return &Queue{
		DB:     db,
		sender: sender,
		Secret: secret,
		poke:   make(chan struct{}, 1),
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}
}

func (q *Queue) SetSender(s Sender) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.sender = s
}

func (q *Queue) senderOrSMTP() Sender {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.sender != nil {
		return q.sender
	}
	return &SMTP{DB: q.DB, Key: q.ConfigKey}
}

func (q *Queue) Poke() {
	if q == nil {
		return
	}
	select {
	case q.poke <- struct{}{}:
	default:
	}
}

func (q *Queue) Stop() {
	if q == nil {
		return
	}
	select {
	case <-q.stop:
		return
	default:
		close(q.stop)
	}
	if q.running.Load() {
		<-q.done
	}
}

func (q *Queue) Run() {
	if q == nil {
		return
	}
	q.running.Store(true)
	defer close(q.done)
	ticker := time.NewTicker(defaultTick)
	defer ticker.Stop()
	for {
		cfg, _ := Load(q.DB, q.ConfigKey)
		d := cfg.WorkerTick
		if d < 200*time.Millisecond {
			d = defaultTick
		}
		ticker.Reset(d)
		select {
		case <-q.stop:
			return
		case <-q.poke:
			now := time.Now()
			q.ReclaimStuck(now)
			q.ProcessCampaigns(now)
			q.ProcessAvailable(now, 32)
		case <-ticker.C:
			now := time.Now()
			q.ReclaimStuck(now)
			q.ProcessCampaigns(now)
			q.ProcessAvailable(now, 32)
		}
	}
}

const stuckSendingAfter = 5 * time.Minute

func (q *Queue) ReclaimStuck(now time.Time) int {
	if q == nil || q.DB == nil {
		return 0
	}
	res := q.DB.Model(&models.MailJob{}).
		Where("status = ? AND updated_at < ?", models.MailStatusSending, now.Add(-stuckSendingAfter)).
		Updates(map[string]any{
			"status":     models.MailStatusQueued,
			"last_error": "reclaimed stuck sending",
			"send_after": now,
		})
	if res.Error != nil || res.RowsAffected == 0 {
		return 0
	}
	return int(res.RowsAffected)
}

type EnqueueInput struct {
	Class      string
	User       *models.User
	ToEmail    string
	Subject    string
	Body       string
	CampaignID *uint
	DedupeKey  string
	Now        time.Time
}

func (q *Queue) Enqueue(in EnqueueInput) (*models.MailJob, error) {
	if q == nil || q.DB == nil {
		return nil, errors.New("mail queue unavailable")
	}
	if in.Now.IsZero() {
		in.Now = time.Now()
	}
	to := strings.TrimSpace(in.ToEmail)
	if in.User != nil && to == "" {
		to = in.User.Email
	}
	if !ValidAddress(to) {
		return nil, ErrInvalidAddr
	}
	class := in.Class
	if class == "" {
		class = models.MailClassOperational
	}
	cfg, err := Load(q.DB, q.ConfigKey)
	if err != nil {
		return nil, err
	}
	tz := cfg.DefaultTimezone
	var uid *uint
	if in.User != nil {
		id := in.User.ID
		uid = &id
		tz = ResolveTimezone(in.User.Timezone, cfg.DefaultTimezone)
	}
	if in.DedupeKey != "" {
		var existing models.MailJob
		err := q.DB.Where("dedupe_key = ? AND status IN ?", in.DedupeKey, []string{
			models.MailStatusQueued, models.MailStatusSending, models.MailStatusSent,
		}).Order("id desc").First(&existing).Error
		if err == nil {
			return &existing, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	userKind := ""
	if in.User != nil {
		userKind = models.NormalizeUserKind(in.User.Kind)
		if userKind == "" {
			userKind = models.UserKindWeb
		}
	}
	job := models.MailJob{
		CampaignID: in.CampaignID,
		Class:      class,
		Priority:   PriorityFor(class),
		UserID:     uid,
		UserKind:   userKind,
		ToEmail:    to,
		Timezone:   tz,
		Subject:    in.Subject,
		Body:       in.Body,
		Status:     models.MailStatusQueued,
		SendAfter:  NextAllowed(in.Now, tz, class, cfg),
		DedupeKey:  in.DedupeKey,
	}
	if err := q.DB.Create(&job).Error; err != nil {
		return nil, err
	}
	if class == models.MailClassTransactional {
		q.Poke()
	}
	return &job, nil
}

func (q *Queue) ProcessAvailable(now time.Time, max int) int {
	if q == nil || max <= 0 {
		return 0
	}
	cfg, err := Load(q.DB, q.ConfigKey)
	if err != nil || !cfg.Enabled {
		return 0
	}
	budget := cfg.RatePerMinute - q.sentLastMinute(now)
	if budget <= 0 {
		return 0
	}
	if max < budget {
		budget = max
	}
	n := 0
	for i := 0; i < budget; i++ {
		if !q.processOne(now, cfg) {
			break
		}
		n++
	}
	return n
}

func (q *Queue) sentLastMinute(now time.Time) int {
	var n int64
	_ = q.DB.Model(&models.MailJob{}).
		Where("status = ? AND sent_at >= ?", models.MailStatusSent, now.Add(-time.Minute)).
		Count(&n).Error
	return int(n)
}

func (q *Queue) processOne(now time.Time, cfg Settings) bool {
	var job models.MailJob
	err := q.DB.Where("status = ? AND send_after <= ?", models.MailStatusQueued, now).
		Order("priority asc, id asc").
		First(&job).Error
	if err != nil {
		return false
	}
	res := q.DB.Model(&models.MailJob{}).
		Where("id = ? AND status = ?", job.ID, models.MailStatusQueued).
		Update("status", models.MailStatusSending)
	if res.Error != nil || res.RowsAffected != 1 {
		return true
	}
	next := NextAllowed(now, job.Timezone, job.Class, cfg)
	if next.After(now.Add(2 * time.Second)) {
		_ = q.DB.Model(&job).Updates(map[string]any{
			"status":     models.MailStatusQueued,
			"send_after": next,
		}).Error
		return true
	}
	body := job.Body
	if job.Class == models.MailClassMarketing && job.UserID != nil && q.Secret != "" {
		kind := job.UserKind
		if kind == "" {
			kind = models.UserKindWeb
		}
		body = appendUnsubFooter(body, UnsubLink(cfg.ResetBaseURL, q.Secret, kind, *job.UserID))
	}
	sendErr := q.senderOrSMTP().Send(job.ToEmail, job.Subject, body)
	if sendErr != nil {
		attempts := job.Attempts + 1
		status := models.MailStatusQueued
		sendAfter := now.Add(backoff(attempts))
		if attempts >= cfg.MaxAttempts {
			status = models.MailStatusDead
		}
		_ = q.DB.Model(&job).Updates(map[string]any{
			"status":     status,
			"attempts":   attempts,
			"last_error": sendErr.Error(),
			"send_after": sendAfter,
		}).Error
		slog.Error("mail job failed", "id", job.ID, "error", sendErr)
		return true
	}
	ts := now
	_ = q.DB.Model(&job).Updates(map[string]any{
		"status":     models.MailStatusSent,
		"sent_at":    ts,
		"last_error": "",
	}).Error
	return true
}

func (q *Queue) Retry(id uint) error {
	var job models.MailJob
	if err := q.DB.First(&job, id).Error; err != nil {
		return err
	}
	if job.Status != models.MailStatusDead && job.Status != models.MailStatusFailed && job.Status != models.MailStatusCanceled {
		return ErrCannotRetry
	}
	return q.DB.Model(&job).Updates(map[string]any{
		"status":     models.MailStatusQueued,
		"send_after": time.Now(),
		"attempts":   0,
		"last_error": "",
	}).Error
}

func (q *Queue) Cancel(id uint) error {
	res := q.DB.Model(&models.MailJob{}).
		Where("id = ? AND status = ?", id, models.MailStatusQueued).
		Update("status", models.MailStatusCanceled)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (q *Queue) CancelCampaign(id uint) error {
	if q == nil {
		return errors.New("mail queue unavailable")
	}
	return q.DB.Model(&models.MailJob{}).
		Where("campaign_id = ? AND status = ?", id, models.MailStatusQueued).
		Update("status", models.MailStatusCanceled).Error
}
