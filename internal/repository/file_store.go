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
}

type FileStore struct {
	mu   sync.RWMutex
	path string
	data map[string]string
}

func NewFileStore(path string) (*FileStore, error) {
	fs := &FileStore{
		path: path,
		data: make(map[string]string),
	}

	if err := fs.load(); err != nil {
		return nil, err
	}

	return fs, nil
}

func (f *FileStore) Save(_ context.Context, id string, original string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.data[id]; exists {
		return ErrIDExists
	}

	f.data[id] = original
	if err := f.flush(); err != nil {
		delete(f.data, id)
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
	}

	for _, item := range items {
		f.data[item.ID] = item.Original
	}

	if err := f.flush(); err != nil {
		for _, item := range items {
			delete(f.data, item.ID)
		}
		return err
	}

	return nil
}

func (f *FileStore) Get(_ context.Context, id string) (string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	v, ok := f.data[id]
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
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
	}

	return nil
}

func (f *FileStore) flush() error {
	records := make([]fileRecord, 0, len(f.data))
	i := 1
	for shortURL, originalURL := range f.data {
		records = append(records, fileRecord{
			UUID:        strconv.Itoa(i),
			ShortURL:    shortURL,
			OriginalURL: originalURL,
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
