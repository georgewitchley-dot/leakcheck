package main

import (
	"context"
	"sync"
	"time"
)

// MemStore is an in-memory Store used by tests and the `nopg` dev build.
type MemStore struct {
	mu     sync.Mutex
	ranges map[string]memEntry
	checks int64
}

type memEntry struct {
	body string
	at   time.Time
}

func NewMemStore() *MemStore { return &MemStore{ranges: map[string]memEntry{}} }

func (m *MemStore) GetRange(_ context.Context, p string) (string, time.Time, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.ranges[p]
	return e.body, e.at, ok, nil
}

func (m *MemStore) PutRange(_ context.Context, p, b string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ranges[p] = memEntry{b, time.Now()}
	return nil
}

func (m *MemStore) IncrementChecks(context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checks++
	return nil
}

func (m *MemStore) TotalChecks(context.Context) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.checks, nil
}

func (m *MemStore) Ping(context.Context) error { return nil }
func (m *MemStore) Close()                     {}
