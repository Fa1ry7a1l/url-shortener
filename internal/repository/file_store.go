package repository

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

type fileRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id,omitempty"`
	IsDeleted   bool   `json:"is_deleted"`
}

type FileStore struct {
	mu      sync.RWMutex
	path    string
	data    map[string]string
	userIDs map[string]string
	deleted map[string]struct{}
}

func NewFileStore(path string) (*FileStore, error) {
	fs := &FileStore{
		path:    path,
		data:    make(map[string]string),
		userIDs: make(map[string]string),
		deleted: make(map[string]struct{}),
	}

	if err := fs.load(); err != nil {
		return nil, err
	}

	return fs, nil
}

func (f *FileStore) Save(ctx context.Context, id string, original string) error {
	return f.SaveForUser(ctx, id, original, "")
}

func (f *FileStore) SaveForUser(_ context.Context, id string, original string, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.data[id]; exists {
		return ErrIDExists
	}

	for _, existingOriginal := range f.data {
		if existingOriginal == original {
			return ErrOriginalURLExist
		}
	}

	f.data[id] = original
	f.userIDs[id] = userID
	if err := f.flush(); err != nil {
		delete(f.data, id)
		delete(f.userIDs, id)
		delete(f.deleted, id)
		return err
	}

	return nil
}

func (f *FileStore) SaveBatch(_ context.Context, items []BatchItem) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, item := range items {
		if _, exists := f.data[item.ID]; exists {
			return ErrIDExists
		}
		for _, existingOriginal := range f.data {
			if existingOriginal == item.Original {
				return ErrOriginalURLExist
			}
		}
	}

	for _, item := range items {
		f.data[item.ID] = item.Original
		f.userIDs[item.ID] = item.UserID
	}

	if err := f.flush(); err != nil {
		for _, item := range items {
			delete(f.data, item.ID)
			delete(f.userIDs, item.ID)
			delete(f.deleted, item.ID)
		}
		return err
	}

	return nil
}

func (f *FileStore) GetByOriginal(_ context.Context, original string) (string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	for id, existingOriginal := range f.data {
		if existingOriginal == original {
			return id, nil
		}
	}

	return "", ErrNotFound
}

func (f *FileStore) Get(_ context.Context, id string) (string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	v, ok := f.data[id]
	if !ok {
		return "", ErrNotFound
	}
	if _, deleted := f.deleted[id]; deleted {
		return "", ErrDeleted
	}
	return v, nil
}

func (f *FileStore) DeleteBatchByUser(_ context.Context, userID string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	hasChanges := false
	for _, id := range ids {
		if f.userIDs[id] != userID {
			continue
		}
		if _, exists := f.data[id]; !exists {
			continue
		}
		if _, deleted := f.deleted[id]; deleted {
			continue
		}
		f.deleted[id] = struct{}{}
		hasChanges = true
	}

	if !hasChanges {
		return nil
	}

	return f.flush()
}

func (f *FileStore) GetByUser(_ context.Context, userID string) ([]UserURL, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	count := 0
	for id, existingUserID := range f.userIDs {
		if existingUserID != userID {
			continue
		}
		if _, deleted := f.deleted[id]; deleted {
			continue
		}
		count++
	}
	if count == 0 {
		return nil, nil
	}

	result := make([]UserURL, 0, count)
	for id, existingUserID := range f.userIDs {
		if existingUserID != userID {
			continue
		}
		if _, deleted := f.deleted[id]; deleted {
			continue
		}
		result = append(result, UserURL{
			ID:       id,
			Original: f.data[id],
		})
	}

	return result, nil
}

func (f *FileStore) Ping(_ context.Context) error {
	return nil
}

func (f *FileStore) load() error {
	if _, err := os.Stat(f.path); errors.Is(err, os.ErrNotExist) {
		return nil
	}

	data, err := os.ReadFile(f.path)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}

	var records []fileRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return err
	}

	for _, rec := range records {
		f.data[rec.ShortURL] = rec.OriginalURL
		f.userIDs[rec.ShortURL] = rec.UserID
		if rec.IsDeleted {
			f.deleted[rec.ShortURL] = struct{}{}
		}
	}

	return nil
}

func (f *FileStore) flush() error {
	records := make([]fileRecord, 0, len(f.data))
	i := 1
	for shortURL, originalURL := range f.data {
		_, deleted := f.deleted[shortURL]
		records = append(records, fileRecord{
			UUID:        strconv.Itoa(i),
			ShortURL:    shortURL,
			OriginalURL: originalURL,
			UserID:      f.userIDs[shortURL],
			IsDeleted:   deleted,
		})
		i++
	}

	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(f.path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	return os.WriteFile(f.path, data, 0o644)
}
