package mailer

import (
	"fmt"
	"strings"
	"time"

	"go-react-shadcn/internal/models"
)

func NormalizeTimezone(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return DefaultTimezone, nil
	}
	if _, err := time.LoadLocation(s); err != nil {
		return "", fmt.Errorf("invalid timezone")
	}
	return s, nil
}

func ResolveTimezone(userTZ, fallback string) string {
	if tz, err := NormalizeTimezone(userTZ); err == nil {
		return tz
	}
	if tz, err := NormalizeTimezone(fallback); err == nil {
		return tz
	}
	return DefaultTimezone
}

func PriorityFor(class string) int {
	switch class {
	case models.MailClassTransactional:
		return models.MailPriorityUrgent
	case models.MailClassMarketing:
		return models.MailPriorityLow
	default:
		return models.MailPriorityNormal
	}
}

// NextAllowed returns the earliest UTC instant when a job of class may be sent.
func NextAllowed(now time.Time, timezone, class string, cfg Settings) time.Time {
	if class == models.MailClassTransactional {
		return now
	}
	loc, err := time.LoadLocation(ResolveTimezone(timezone, cfg.DefaultTimezone))
	if err != nil {
		loc = time.UTC
	}
	local := now.In(loc)
	if class == models.MailClassMarketing {
		if inClockWindow(local, cfg.MarketingStart, cfg.MarketingEnd) && !inClockWindow(local, cfg.QuietStart, cfg.QuietEnd) {
			return now
		}
		return nextClock(local, cfg.MarketingStart).UTC()
	}
	if !inClockWindow(local, cfg.QuietStart, cfg.QuietEnd) {
		return now
	}
	return nextClock(local, cfg.QuietEnd).UTC()
}

func inClockWindow(local time.Time, start, end string) bool {
	clock := local.Hour()*60 + local.Minute()
	s := parseClock(start)
	e := parseClock(end)
	if s == e {
		return true
	}
	if s < e {
		return clock >= s && clock < e
	}
	return clock >= s || clock < e
}

func parseClock(hm string) int {
	var h, m int
	_, _ = fmt.Sscanf(strings.TrimSpace(hm), "%d:%d", &h, &m)
	if h < 0 {
		h = 0
	}
	if h > 23 {
		h = 23
	}
	if m < 0 {
		m = 0
	}
	if m > 59 {
		m = 59
	}
	return h*60 + m
}

func nextClock(local time.Time, hm string) time.Time {
	mins := parseClock(hm)
	candidate := time.Date(local.Year(), local.Month(), local.Day(), mins/60, mins%60, 0, 0, local.Location())
	if !local.Before(candidate) {
		candidate = candidate.Add(24 * time.Hour)
	}
	return candidate
}

func backoff(attempts int) time.Duration {
	if attempts < 1 {
		attempts = 1
	}
	d := time.Duration(1<<(attempts-1)) * time.Minute
	if d > time.Hour {
		return time.Hour
	}
	return d
}
