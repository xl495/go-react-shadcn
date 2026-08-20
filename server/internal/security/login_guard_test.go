package security

import (
	"testing"
	"time"
)

func TestAllowIPEvictsIdleKeys(t *testing.T) {
	g := NewLoginGuard()
	g.ipWindow = 20 * time.Millisecond
	g.ipLimit = 5
	if !g.AllowIP("1.1.1.1") {
		t.Fatal("first hit should pass")
	}
	if g.TrackedIPs() != 1 {
		t.Fatalf("tracked=%d", g.TrackedIPs())
	}
	time.Sleep(30 * time.Millisecond)
	if !g.AllowIP("2.2.2.2") {
		t.Fatal("other ip should pass")
	}
	g.Sweep()
	if g.TrackedIPs() != 1 {
		t.Fatalf("after sweep tracked=%d want 1", g.TrackedIPs())
	}
}
