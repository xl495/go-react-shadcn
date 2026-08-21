package captcha

import (
	"strings"
	"time"

	"github.com/mojocn/base64Captcha"
	"gorm.io/gorm"
)

const ttl = 3 * time.Minute

type Challenge struct {
	ID     string `json:"captchaId"`
	Image  string `json:"image"`
	Answer string `json:"answer,omitempty"`
}

type Service struct {
	engine *base64Captcha.Captcha
	store  base64Captcha.Store
	debug  bool
}

type challengeRow struct {
	ID        string    `gorm:"primaryKey;size:64"`
	Answer    string    `gorm:"size:32;not null"`
	ExpiresAt time.Time `gorm:"index;not null"`
}

func (challengeRow) TableName() string { return "captcha_challenges" }

type sqlStore struct {
	db *gorm.DB
}

func New(db *gorm.DB, debug bool) *Service {
	var store base64Captcha.Store
	if db == nil {
		store = base64Captcha.NewMemoryStore(1024, ttl)
	} else {
		store = newSQLStore(db)
	}
	driver := base64Captcha.NewDriverDigit(80, 260, 6, 0.7, 90)
	return &Service{
		engine: base64Captcha.NewCaptcha(driver, store),
		store:  store,
		debug:  debug,
	}
}

func newSQLStore(db *gorm.DB) *sqlStore {
	return &sqlStore{db: db}
}

func (s *sqlStore) Set(id string, value string) error {
	if s == nil || s.db == nil || id == "" {
		return nil
	}
	now := time.Now()
	if err := s.db.Where("expires_at < ?", now).Delete(&challengeRow{}).Error; err != nil {
		return err
	}
	return s.db.Save(&challengeRow{ID: id, Answer: value, ExpiresAt: now.Add(ttl)}).Error
}

func (s *sqlStore) Get(id string, clear bool) string {
	if s == nil || s.db == nil || id == "" {
		return ""
	}
	var row challengeRow
	err := s.db.Where("id = ? AND expires_at > ?", id, time.Now()).First(&row).Error
	if err != nil {
		return ""
	}
	if clear {
		_ = s.db.Delete(&challengeRow{}, "id = ?", id).Error
	}
	return row.Answer
}

func (s *sqlStore) Verify(id, answer string, clear bool) bool {
	got := s.Get(id, clear)
	return got != "" && got == strings.TrimSpace(answer)
}

func (s *Service) Issue() (Challenge, error) {
	id, b64, _, err := s.engine.Generate()
	if err != nil {
		return Challenge{}, err
	}
	if !strings.HasPrefix(b64, "data:") {
		b64 = "data:image/png;base64," + b64
	}
	ch := Challenge{ID: id, Image: b64}
	if s.debug {
		ch.Answer = s.Peek(id)
	}
	return ch, nil
}

func (s *Service) Verify(id, answer string) bool {
	if id == "" || strings.TrimSpace(answer) == "" {
		return false
	}
	return s.store.Verify(id, strings.TrimSpace(answer), true)
}

func (s *Service) Peek(id string) string {
	return s.store.Get(id, false)
}
