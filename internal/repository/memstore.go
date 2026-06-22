package repository

import (
	"context"
	"sync"
)

type MemStore struct {
	mu      sync.RWMutex
	data    map[string]string
	userIDs map[string]string
	deleted map[string]struct{}
}

func NewMemStore() *MemStore {
	return &MemStore{
		data:    make(map[string]string),
		userIDs: make(map[string]string),
		deleted: make(map[string]struct{}),
	}
}

func (m *MemStore) Save(ctx context.Context, id string, original string) error {
	return m.SaveForUser(ctx, id, original, "")
}

func (m *MemStore) SaveForUser(_ context.Context, id string, original string, userID string) error {
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
	m.userIDs[id] = userID
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
		m.userIDs[item.ID] = item.UserID
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
	if _, deleted := m.deleted[id]; deleted {
		return "", ErrDeleted
	}

	return original, nil
}

func (m *MemStore) DeleteBatchByUser(_ context.Context, userID string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, id := range ids {
		if m.userIDs[id] != userID {
			continue
		}
		if _, exists := m.data[id]; !exists {
			continue
		}
		m.deleted[id] = struct{}{}
	}

	return nil
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

func (m *MemStore) GetByUser(_ context.Context, userID string) ([]UserURL, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for id, existingUserID := range m.userIDs {
		if existingUserID != userID {
			continue
		}
		if _, deleted := m.deleted[id]; deleted {
			continue
		}
		count++
	}
	if count == 0 {
		return nil, nil
	}

	result := make([]UserURL, 0, count)
	for id, existingUserID := range m.userIDs {
		if existingUserID != userID {
			continue
		}
		if _, deleted := m.deleted[id]; deleted {
			continue
		}
		result = append(result, UserURL{
			ID:       id,
			Original: m.data[id],
		})
	}

	return result, nil
}

func (m *MemStore) Ping(_ context.Context) error {
	return nil
}
