package repository

import (
	"context"
	"sync"
)

type MemStore struct {
	mu   sync.RWMutex
	data map[string]string
}

func NewMemStore() *MemStore {
	return &MemStore{
		data: make(map[string]string),
	}
}

func (m *MemStore) Save(_ context.Context, id string, original string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.data[id]; exists {
		return ErrIDExists
	}

	for _, existingOriginal := range m.data {
		if existingOriginal == original {
			return ErrOriginalURLExist
		}
	}

	m.data[id] = original
	return nil
}

func (m *MemStore) SaveBatch(_ context.Context, items []BatchItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, item := range items {
		if _, exists := m.data[item.ID]; exists {
			return ErrIDExists
		}
		for _, existingOriginal := range m.data {
			if existingOriginal == item.Original {
				return ErrOriginalURLExist
			}
		}
	}

	for _, item := range items {
		m.data[item.ID] = item.Original
	}

	return nil
}

func (m *MemStore) Get(_ context.Context, id string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	original, ok := m.data[id]
	if !ok {
		return "", ErrNotFound
	}

	return original, nil
}

func (m *MemStore) GetByOriginal(_ context.Context, original string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for id, existingOriginal := range m.data {
		if existingOriginal == original {
			return id, nil
		}
	}

	return "", ErrNotFound
}

func (m *MemStore) Ping(_ context.Context) error {
	return nil
}
