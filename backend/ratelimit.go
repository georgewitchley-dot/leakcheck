package main

import (
	"sync"
	"time"
)

// RateLimiter is a fixed-window per-key limiter held in memory. IPs are used
// only as transient map keys and are never logged or persisted.
type RateLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	hits   map[string]*bucket
	last   time.Time
}

type bucket struct {
	start time.Time
	n     int
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{limit: limit, window: window, hits: map[string]*bucket{}, last: time.Now()}
}

func (l *RateLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if now.Sub(l.last) > l.window {
		for k, b := range l.hits {
			if now.Sub(b.start) > l.window {
				delete(l.hits, k)
			}
		}
		l.last = now
	}
	b, ok := l.hits[key]
	if !ok || now.Sub(b.start) > l.window {
		l.hits[key] = &bucket{start: now, n: 1}
		return true
	}
	if b.n >= l.limit {
		return false
	}
	b.n++
	return true
}
