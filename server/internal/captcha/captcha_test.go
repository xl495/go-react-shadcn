package captcha

import (
	"path/filepath"
	"testing"

	"go-react-shadcn/internal/migrate"
	"go-react-shadcn/internal/store"
)

func TestSQLStoreSharedAcrossServices(t *testing.T) {
	path := filepath.Join(t.TempDir(), "captcha.db")
	if err := migrate.Up(path); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	a := New(db, true)
	b := New(db, false)
	ch, err := a.Issue()
	if err != nil {
		t.Fatal(err)
	}
	if ch.Answer == "" || len(ch.Answer) != 6 {
		t.Fatalf("answer=%q", ch.Answer)
	}
	if !b.Verify(ch.ID, ch.Answer) {
		t.Fatal("second service should verify the same challenge")
	}
	if b.Verify(ch.ID, ch.Answer) {
		t.Fatal("challenge should be one-shot")
	}
}
