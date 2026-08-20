package mailer

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"sync"
	"time"

	"go-react-shadcn/internal/models"
	"gorm.io/gorm"
)

const (
	ResetTTL         = 30 * time.Minute
	SecretMask       = "********"
	defaultPort      = 587
	dialTimeout      = 10 * time.Second
	defaultFromNm    = "Latch"
	DefaultTimezone  = "Asia/Shanghai"
	defaultQuietStart = "22:00"
	defaultQuietEnd   = "08:00"
	defaultMktStart   = "09:00"
	defaultMktEnd     = "21:00"
	defaultRate       = 30
	defaultAttempts   = 5
	defaultTick       = 2 * time.Second
)

var (
	ErrDisabled    = errors.New("mail is not enabled")
	ErrIncomplete  = errors.New("mail is not configured")
	ErrInvalidAddr = errors.New("invalid email address")
)

type Settings struct {
	Enabled          bool
	Host             string
	Port             int
	Username         string
	Password         string
	From             string
	FromName         string
	TLS              string
	ResetBaseURL     string
	DefaultTimezone  string
	QuietStart       string
	QuietEnd         string
	MarketingStart   string
	MarketingEnd     string
	RatePerMinute    int
	MaxAttempts      int
	WorkerTick       time.Duration
}

type Sender interface {
	Send(to, subject, body string) error
}

type SMTP struct {
	DB *gorm.DB
}

func (s *SMTP) Send(to, subject, body string) error {
	cfg, err := Load(s.DB)
	if err != nil {
		return err
	}
	return Send(cfg, to, subject, body)
}

type Message struct {
	To      string
	Subject string
	Body    string
}

type Memory struct {
	mu         sync.Mutex
	Mails      []Message
	Err        error
	Disabled   bool
	Incomplete bool
}

func (m *Memory) Send(to, subject, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Disabled {
		return ErrDisabled
	}
	if m.Incomplete {
		return ErrIncomplete
	}
	if m.Err != nil {
		return m.Err
	}
	m.Mails = append(m.Mails, Message{To: to, Subject: subject, Body: body})
	return nil
}

func (m *Memory) Last() (Message, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.Mails) == 0 {
		return Message{}, false
	}
	return m.Mails[len(m.Mails)-1], true
}

func Load(db *gorm.DB) (Settings, error) {
	var rows []models.SysConfig
	if err := db.Where("`group` = ?", "mail").Find(&rows).Error; err != nil {
		return Settings{}, err
	}
	vals := map[string]string{}
	for _, row := range rows {
		vals[row.Key] = row.Value
	}
	port, _ := strconv.Atoi(strings.TrimSpace(vals["mail.port"]))
	if port <= 0 {
		port = defaultPort
	}
	fromName := strings.TrimSpace(vals["mail.from_name"])
	if fromName == "" {
		fromName = defaultFromNm
	}
	enabled := vals["mail.enabled"] == "1" || strings.EqualFold(vals["mail.enabled"], "true")
	tlsMode := strings.ToLower(strings.TrimSpace(vals["mail.tls"]))
	if tlsMode == "" {
		tlsMode = "starttls"
	}
	return Settings{
		Enabled:         enabled,
		Host:            strings.TrimSpace(vals["mail.host"]),
		Port:            port,
		Username:        strings.TrimSpace(vals["mail.username"]),
		Password:        vals["mail.password"],
		From:            strings.TrimSpace(vals["mail.from"]),
		FromName:        fromName,
		TLS:             tlsMode,
		ResetBaseURL:    strings.TrimRight(strings.TrimSpace(vals["mail.reset_base_url"]), "/"),
		DefaultTimezone: firstNonEmpty(vals["mail.default_timezone"], DefaultTimezone),
		QuietStart:      firstNonEmpty(vals["mail.quiet_start"], defaultQuietStart),
		QuietEnd:        firstNonEmpty(vals["mail.quiet_end"], defaultQuietEnd),
		MarketingStart:  firstNonEmpty(vals["mail.marketing_start"], defaultMktStart),
		MarketingEnd:    firstNonEmpty(vals["mail.marketing_end"], defaultMktEnd),
		RatePerMinute:   atoiDefault(vals["mail.rate_per_minute"], defaultRate),
		MaxAttempts:     atoiDefault(vals["mail.max_attempts"], defaultAttempts),
		WorkerTick:      time.Duration(atoiDefault(vals["mail.worker_tick_ms"], int(defaultTick/time.Millisecond))) * time.Millisecond,
	}, nil
}

func firstNonEmpty(v, fallback string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return fallback
	}
	return v
}

func atoiDefault(v string, fallback int) int {
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func Send(cfg Settings, to, subject, body string) error {
	if !cfg.Enabled {
		return ErrDisabled
	}
	if cfg.Host == "" || cfg.From == "" {
		return ErrIncomplete
	}
	if !ValidAddress(to) || !ValidAddress(cfg.From) {
		return ErrInvalidAddr
	}
	msg := buildMessage(cfg, to, subject, body)
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	var auth smtp.Auth
	if cfg.Username != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	}
	switch cfg.TLS {
	case "ssl", "tls":
		return sendSMTPS(addr, cfg.Host, auth, cfg.From, to, msg)
	default:
		return smtp.SendMail(addr, auth, cfg.From, []string{to}, msg)
	}
}

func buildMessage(cfg Settings, to, subject, body string) []byte {
	from := fmt.Sprintf("%s <%s>", mime.QEncoding.Encode("UTF-8", cfg.FromName), cfg.From)
	contentType := "text/plain; charset=UTF-8"
	if LooksHTML(body) {
		contentType = "text/html; charset=UTF-8"
		body = wrapHTML(body)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", from)
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Subject: %s\r\n", mime.BEncoding.Encode("UTF-8", subject))
	b.WriteString("MIME-Version: 1.0\r\n")
	fmt.Fprintf(&b, "Content-Type: %s\r\n", contentType)
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	b.WriteString(body)
	if !strings.HasSuffix(body, "\n") {
		b.WriteString("\r\n")
	}
	return []byte(b.String())
}

func sendSMTPS(addr, host string, auth smtp.Auth, from, to string, msg []byte) error {
	dialer := net.Dialer{Timeout: dialTimeout}
	conn, err := tls.DialWithDialer(&dialer, "tcp", addr, &tls.Config{
		ServerName: host,
		MinVersion: tls.VersionTLS12,
	})
	if err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		return err
	}
	defer func() { _ = client.Close() }()
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		_ = w.Close()
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func ValidAddress(s string) bool {
	s = strings.TrimSpace(s)
	at := strings.IndexByte(s, '@')
	return at > 0 && at < len(s)-1 && !strings.ContainsAny(s, " \t\r\n")
}

func LooksSecret(key string) bool {
	k := strings.ToLower(key)
	return strings.Contains(k, "password") || strings.Contains(k, "secret")
}

func Redact(row models.SysConfig) models.SysConfig {
	if LooksSecret(row.Key) && row.Value != "" {
		row.Value = SecretMask
	}
	return row
}

func KeepSecret(key, incoming, current string) string {
	if !LooksSecret(key) {
		return incoming
	}
	if incoming == "" || incoming == SecretMask {
		return current
	}
	return incoming
}

func NewResetToken() (raw, hash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(buf)
	return raw, HashToken(raw), nil
}

func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func ResetLink(baseURL, origin, token string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = strings.TrimRight(strings.TrimSpace(origin), "/")
	}
	if base == "" {
		base = "http://127.0.0.1:5173"
	}
	return base + "/reset-password?token=" + token
}
