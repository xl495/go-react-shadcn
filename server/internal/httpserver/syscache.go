package httpserver

import (
	"sync"
	"time"
)

type sysCacheEntry struct {
	value   string
	missing bool
	expires time.Time
}

type sysCache struct {
	mu    sync.Mutex
	ttl   time.Duration
	items map[string]sysCacheEntry
}

func newSysCache(ttl time.Duration) *sysCache {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &sysCache{ttl: ttl, items: map[string]sysCacheEntry{}}
}

func (s *sysCache) get(key string) (string, bool) {
	if s == nil {
		return "", false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.items[key]
	if !ok || time.Now().After(entry.expires) {
		delete(s.items, key)
		return "", false
	}
	if entry.missing {
		return "", true
	}
	return entry.value, true
}

func (s *sysCache) put(key, value string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = sysCacheEntry{value: value, missing: value == "", expires: time.Now().Add(s.ttl)}
}

func (s *sysCache) invalidate() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = map[string]sysCacheEntry{}
}
