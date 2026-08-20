package mailer

import (
	"testing"
	"time"

	"go-react-shadcn/internal/models"
)

func TestNextAllowedTransactionalIgnoresWindow(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 20, 23, 30, 0, 0, loc)
	cfg := Settings{
		DefaultTimezone: "Asia/Shanghai",
		QuietStart:      "22:00",
		QuietEnd:        "08:00",
		MarketingStart:  "09:00",
		MarketingEnd:    "21:00",
	}
	got := NextAllowed(now, "Asia/Shanghai", models.MailClassTransactional, cfg)
	if !got.Equal(now) {
		t.Fatalf("transactional deferred: %s", got)
	}
}

func TestNextAllowedMarketingAfterQuiet(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 20, 23, 30, 0, 0, loc)
	cfg := Settings{
		DefaultTimezone: "Asia/Shanghai",
		QuietStart:      "22:00",
		QuietEnd:        "08:00",
		MarketingStart:  "09:00",
		MarketingEnd:    "21:00",
	}
	got := NextAllowed(now, "Asia/Shanghai", models.MailClassMarketing, cfg).In(loc)
	want := time.Date(2026, 8, 21, 9, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestNextAllowedOperationalDuringQuiet(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 20, 23, 0, 0, 0, loc)
	cfg := Settings{
		DefaultTimezone: "Asia/Shanghai",
		QuietStart:      "22:00",
		QuietEnd:        "08:00",
		MarketingStart:  "09:00",
		MarketingEnd:    "21:00",
	}
	got := NextAllowed(now, "Asia/Tokyo", models.MailClassOperational, cfg).In(loc)
	want := time.Date(2026, 8, 21, 8, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestNormalizeTimezone(t *testing.T) {
	got, err := NormalizeTimezone("")
	if err != nil || got != DefaultTimezone {
		t.Fatalf("empty: %q %v", got, err)
	}
	if _, err := NormalizeTimezone("Not/AZone"); err == nil {
		t.Fatal("expected invalid tz")
	}
}
