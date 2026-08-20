package httpserver

import (
	"sync"
	"time"
)

type sessionSnapshot struct {
	tokenVersion int
	status       string
	expires      time.Time
}

type sessionCache struct {
	mu    sync.Mutex
	ttl   time.Duration
	items map[uint]sessionSnapshot
}

func newSessionCache(ttl time.Duration) *sessionCache {
	if ttl <= 0 {
		ttl = 2 * time.Second
	}
	return &sessionCache{ttl: ttl, items: map[uint]sessionSnapshot{}}
}

func (s *sessionCache) get(id uint) (sessionSnapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap, ok := s.items[id]
	if !ok || time.Now().After(snap.expires) {
		delete(s.items, id)
		return sessionSnapshot{}, false
	}
	return snap, true
}

func (s *sessionCache) put(id uint, tokenVersion int, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[id] = sessionSnapshot{
		tokenVersion: tokenVersion,
		status:       status,
		expires:      time.Now().Add(s.ttl),
	}
}

func (s *sessionCache) invalidate(id uint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, id)
}
