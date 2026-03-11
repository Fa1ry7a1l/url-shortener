package repository

import (
	"context"
	"errors"
	"sync"
)

var ErrNotFound = errors.New("not found")

type URLStore interface {
	Save(ctx context.Context, id string, original string) error
	Get(ctx context.Context, id string) (string, error)
}

type MemStore struct {
	mu   sync.RWMutex
	data map[string]string
}

func NewMemStore() *MemStore {
	return &MemStore{data: make(map[string]string)}
}

func (m *MemStore) Save(_ context.Context, id string, original string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[id] = original
	return nil
}

func (m *MemStore) Get(_ context.Context, id string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	v, ok := m.data[id]
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
}
