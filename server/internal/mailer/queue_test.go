package mailer

import (
	"path/filepath"
	"testing"
	"time"

	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/store"
)

func testQueueDB(t *testing.T) *Queue {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "mail.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(models.AllModels()...); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.SysConfig{Key: "mail.enabled", Value: "1", Name: "e", Group: "mail"}).Error; err != nil {
		t.Fatal(err)
	}
	mem := &Memory{}
	q := NewQueue(db, mem, "test-secret")
	t.Cleanup(q.Stop)
	return q
}

func TestQueueTransactionalSendsNow(t *testing.T) {
	q := testQueueDB(t)
	user := models.User{
		Username: "u1", PasswordHash: "x", Email: "u1@example.com", Status: "active",
		Timezone: "Asia/Shanghai", MarketingOptIn: true,
	}
	if err := q.DB.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	job, err := q.Enqueue(EnqueueInput{
		Class: models.MailClassTransactional, User: &user, Subject: "hi", Body: "body",
	})
	if err != nil {
		t.Fatal(err)
	}
	if n := q.ProcessAvailable(time.Now(), 4); n != 1 {
		t.Fatalf("processed=%d", n)
	}
	mem := q.senderOrSMTP().(*Memory)
	msg, ok := mem.Last()
	if !ok || msg.To != "u1@example.com" {
		t.Fatalf("sent %+v", msg)
	}
	if err := q.DB.First(job, job.ID).Error; err != nil {
		t.Fatal(err)
	}
	if job.Status != models.MailStatusSent {
		t.Fatalf("status=%s", job.Status)
	}
}

func TestQueueMarketingDefersQuietHours(t *testing.T) {
	q := testQueueDB(t)
	if err := q.DB.Create(&models.SysConfig{Key: "mail.quiet_start", Value: "22:00", Name: "q", Group: "mail"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := q.DB.Create(&models.SysConfig{Key: "mail.marketing_start", Value: "09:00", Name: "m", Group: "mail"}).Error; err != nil {
		t.Fatal(err)
	}
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 20, 23, 30, 0, 0, loc)
	job, err := q.Enqueue(EnqueueInput{
		Class:   models.MailClassMarketing,
		ToEmail: "m@example.com",
		Subject: "promo",
		Body:    "sale",
		Now:     now,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 8, 21, 9, 0, 0, 0, loc)
	if !job.SendAfter.Equal(want.UTC()) && !job.SendAfter.In(loc).Equal(want) {
		t.Fatalf("sendAfter=%s want %s", job.SendAfter.In(loc), want)
	}
	if n := q.ProcessAvailable(now, 4); n != 0 {
		t.Fatalf("should not send during quiet, processed=%d", n)
	}
}

func TestQueueDedupe(t *testing.T) {
	q := testQueueDB(t)
	in := EnqueueInput{Class: models.MailClassTransactional, ToEmail: "d@example.com", Subject: "s", Body: "b", DedupeKey: "k1"}
	a, err := q.Enqueue(in)
	if err != nil {
		t.Fatal(err)
	}
	b, err := q.Enqueue(in)
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != b.ID {
		t.Fatalf("dedupe created new job %d vs %d", a.ID, b.ID)
	}
}
