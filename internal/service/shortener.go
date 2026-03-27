package service

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/Fa1ry7a1l/url-shortener/internal/repository"
)

type BatchRequestItem struct {
	CorrelationID string
	OriginalURL   string
}

type BatchResponseItem struct {
	CorrelationID string
	ShortURL      string
}

type ConflictError struct {
	ShortURL string
}

func (e *ConflictError) Error() string {
	return "original url already exists"
}

type Shortener struct {
	store          repository.Store
	idgen          IDGenerator
	baseURL        string
	maxSaveRetries int
}

func NewShortener(store repository.Store, idgen IDGenerator, baseURL string) *Shortener {
	baseURL = strings.TrimRight(baseURL, "/")
	return &Shortener{
		store:          store,
		idgen:          idgen,
		baseURL:        baseURL,
		maxSaveRetries: 10,
	}
}

func (s *Shortener) Shorten(ctx context.Context, original string) (string, error) {
	original = strings.TrimSpace(original)
	if !isValidURL(original) {
		return "", errors.New("invalid url")
	}

	for i := 0; i < s.maxSaveRetries; i++ {
		id, err := s.idgen.NewID()
		if err != nil {
			return "", err
		}

		err = s.store.Save(ctx, id, original)
		if err == nil {
			return s.baseURL + "/" + id, nil
		}

		if errors.Is(err, repository.ErrIDExists) {
			continue
		}

		if errors.Is(err, repository.ErrOriginalURLExist) {
			existingID, getErr := s.store.GetByOriginal(ctx, original)
			if getErr != nil {
				return "", getErr
			}
			return "", &ConflictError{
				ShortURL: s.baseURL + "/" + existingID,
			}
		}

		return "", err
	}

	return "", errors.New("failed to generate unique id")
}

func (s *Shortener) ShortenBatch(ctx context.Context, items []BatchRequestItem) ([]BatchResponseItem, error) {
	if len(items) == 0 {
		return nil, errors.New("empty batch")
	}

	result := make([]BatchResponseItem, 0, len(items))
	storeItems := make([]repository.BatchItem, 0, len(items))
	usedIDs := make(map[string]struct{})

	for _, item := range items {
		original := strings.TrimSpace(item.OriginalURL)
		if item.CorrelationID == "" || !isValidURL(original) {
			return nil, errors.New("invalid batch item")
		}

		var id string
		var err error

		for i := 0; i < s.maxSaveRetries; i++ {
			id, err = s.idgen.NewID()
			if err != nil {
				return nil, err
			}
			if _, exists := usedIDs[id]; exists {
				continue
			}
			usedIDs[id] = struct{}{}
			break
		}

		if id == "" {
			return nil, errors.New("failed to generate unique id")
		}

		storeItems = append(storeItems, repository.BatchItem{
			ID:       id,
			Original: original,
		})
		result = append(result, BatchResponseItem{
			CorrelationID: item.CorrelationID,
			ShortURL:      s.baseURL + "/" + id,
		})
	}

	err := s.store.SaveBatch(ctx, storeItems)
	if err != nil {
		if errors.Is(err, repository.ErrIDExists) {
			// редкая коллизия с уже существующими в storage id
			// для batch просто просим caller повторить запрос
			return nil, errors.New("failed to save batch due to id collision")
		}
		return nil, err
	}

	return result, nil
}

func (s *Shortener) Resolve(ctx context.Context, id string) (string, error) {
	if id == "" || strings.Contains(id, "/") {
		return "", errors.New("invalid id")
	}
	return s.store.Get(ctx, id)
}

func isValidURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if u.Scheme == "" || u.Host == "" {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}
