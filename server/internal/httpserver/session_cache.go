package httpserver

import (
	"strconv"
	"sync"
	"time"

	"go-react-shadcn/internal/models"
	"go-react-shadcn/internal/token"
)

type sessionSnapshot struct {
	tokenVersion int
	status       string
	expires      time.Time
}

type sessionCache struct {
	mu    sync.Mutex
	ttl   time.Duration
	items map[string]sessionSnapshot
}

func newSessionCache(ttl time.Duration) *sessionCache {
	if ttl <= 0 {
		ttl = 2 * time.Second
	}
	return &sessionCache{ttl: ttl, items: map[string]sessionSnapshot{}}
}

func accountSessionKey(kind string, id uint) string {
	return models.NormalizeUserKind(kind) + ":" + strconv.FormatUint(uint64(id), 10)
}

func claimsKind(c *token.Claims) string {
	if c == nil {
		return models.UserKindAdmin
	}
	return models.NormalizeUserKind(c.Kind)
}

func (s *sessionCache) get(kind string, id uint) (sessionSnapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := accountSessionKey(kind, id)
	snap, ok := s.items[key]
	if !ok || time.Now().After(snap.expires) {
		delete(s.items, key)
		return sessionSnapshot{}, false
	}
	return snap, true
}

func (s *sessionCache) put(kind string, id uint, tokenVersion int, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[accountSessionKey(kind, id)] = sessionSnapshot{
		tokenVersion: tokenVersion,
		status:       status,
		expires:      time.Now().Add(s.ttl),
	}
}

func (s *sessionCache) invalidate(kind string, id uint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, accountSessionKey(kind, id))
}

func (s *sessionCache) Sweep() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for k, snap := range s.items {
		if now.After(snap.expires) {
			delete(s.items, k)
		}
	}
}
