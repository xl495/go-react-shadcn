package security

import (
	"sync"
	"time"
)

const (
	DefaultMaxFailures = 5
	DefaultLockMinutes = 15
	DefaultIPLimit     = 30
	DefaultIPWindow    = time.Minute
)

type LoginGuard struct {
	mu          sync.Mutex
	maxFailures int
	lockFor     time.Duration
	ipLimit     int
	ipWindow    time.Duration
	ipHits      map[string][]time.Time
}

func NewLoginGuard() *LoginGuard {
	return &LoginGuard{
		maxFailures: DefaultMaxFailures,
		lockFor:     DefaultLockMinutes * time.Minute,
		ipLimit:     DefaultIPLimit,
		ipWindow:    DefaultIPWindow,
		ipHits:      make(map[string][]time.Time),
	}
}

func (g *LoginGuard) AllowIP(ip string) bool {
	now := time.Now()
	g.mu.Lock()
	defer g.mu.Unlock()
	cutoff := now.Add(-g.ipWindow)
	hits := g.ipHits[ip]
	kept := hits[:0]
	for _, t := range hits {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= g.ipLimit {
		g.ipHits[ip] = kept
		return false
	}
	g.ipHits[ip] = append(kept, now)
	return true
}

func (g *LoginGuard) LockDuration() time.Duration {
	return g.lockFor
}

func (g *LoginGuard) MaxFailures() int {
	return g.maxFailures
}

func (g *LoginGuard) ShouldLock(failCount int) bool {
	return failCount >= g.maxFailures
}

func (g *LoginGuard) LockedUntil(now time.Time, failCount int) *time.Time {
	if !g.ShouldLock(failCount) {
		return nil
	}
	until := now.Add(g.lockFor)
	return &until
}

func IsLocked(until *time.Time, now time.Time) bool {
	return until != nil && until.After(now)
}
