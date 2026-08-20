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
	return NewIPLimiter(DefaultIPLimit, DefaultIPWindow)
}

func NewIPLimiter(limit int, window time.Duration) *LoginGuard {
	if limit <= 0 {
		limit = DefaultIPLimit
	}
	if window <= 0 {
		window = DefaultIPWindow
	}
	return &LoginGuard{
		maxFailures: DefaultMaxFailures,
		lockFor:     DefaultLockMinutes * time.Minute,
		ipLimit:     limit,
		ipWindow:    window,
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
	if len(kept) == 0 {
		delete(g.ipHits, ip)
	} else {
		g.ipHits[ip] = kept
	}
	if len(kept) >= g.ipLimit {
		return false
	}
	g.ipHits[ip] = append(kept, now)
	if len(g.ipHits) > 4096 {
		g.evictExpiredLocked(cutoff)
	}
	return true
}

func (g *LoginGuard) evictExpiredLocked(cutoff time.Time) {
	for k, hits := range g.ipHits {
		alive := hits[:0]
		for _, t := range hits {
			if t.After(cutoff) {
				alive = append(alive, t)
			}
		}
		if len(alive) == 0 {
			delete(g.ipHits, k)
		} else {
			g.ipHits[k] = alive
		}
	}
}

func (g *LoginGuard) Sweep() {
	now := time.Now()
	g.mu.Lock()
	defer g.mu.Unlock()
	g.evictExpiredLocked(now.Add(-g.ipWindow))
}

func (g *LoginGuard) TrackedIPs() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.ipHits)
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
