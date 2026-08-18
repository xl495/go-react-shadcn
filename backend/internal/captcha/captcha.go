package captcha

import (
	"strings"
	"time"

	"github.com/mojocn/base64Captcha"
)

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

func New(debug bool) *Service {
	store := base64Captcha.NewMemoryStore(1024, 3*time.Minute)
	driver := base64Captcha.NewDriverDigit(72, 200, 4, 0.35, 50)
	return &Service{
		engine: base64Captcha.NewCaptcha(driver, store),
		store:  store,
		debug:  debug,
	}
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
