package service

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/Fa1ry7a1l/url-shortener/internal/auth"
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

type UserURL struct {
	ShortURL    string
	OriginalURL string
}

var ErrUnauthorized = errors.New("missing user id")

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
	deleteQueue    chan deleteItem
}

type deleteItem struct {
	userID string
	id     string
}

const (
	deleteQueueSize     = 4096
	deleteBatchSize     = 256
	deleteFlushInterval = 100 * time.Millisecond
)

func NewShortener(store repository.Store, idgen IDGenerator, baseURL string) *Shortener {
	baseURL = strings.TrimRight(baseURL, "/")
	s := &Shortener{
		store:          store,
		idgen:          idgen,
		baseURL:        baseURL,
		maxSaveRetries: 10,
		deleteQueue:    make(chan deleteItem, deleteQueueSize),
	}

	go s.RunDeleteWorker(context.Background())

	return s
}

func (s *Shortener) Shorten(ctx context.Context, original string) (string, error) {
	original = strings.TrimSpace(original)
	if !isValidURL(original) {
		return "", errors.New("invalid url")
	}
	userID, _ := auth.UserIDFromContext(ctx)

	for i := 0; i < s.maxSaveRetries; i++ {
		id, err := s.idgen.NewID()
		if err != nil {
			return "", err
		}

		err = s.store.SaveForUser(ctx, id, original, userID)
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
	userID, _ := auth.UserIDFromContext(ctx)

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
			UserID:   userID,
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

func (s *Shortener) UserURLs(ctx context.Context) ([]UserURL, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return nil, ErrUnauthorized
	}

	items, err := s.store.GetByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]UserURL, 0, len(items))
	for _, item := range items {
		result = append(result, UserURL{
			ShortURL:    s.baseURL + "/" + item.ID,
			OriginalURL: item.Original,
		})
	}

	return result, nil
}

func (s *Shortener) DeleteURLs(ctx context.Context, ids []string) error {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return ErrUnauthorized
	}
	if len(ids) == 0 {
		return errors.New("empty ids")
	}

	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || strings.Contains(id, "/") {
			return errors.New("invalid id")
		}

		select {
		case s.deleteQueue <- deleteItem{userID: userID, id: id}:
		default:
			return errors.New("delete queue is full")
		}
	}

	return nil
}

func (s *Shortener) RunDeleteWorker(ctx context.Context) {
	ticker := time.NewTicker(deleteFlushInterval)
	defer ticker.Stop()

	batch := make([]deleteItem, 0, deleteBatchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}

		grouped := make(map[string][]string)
		for _, item := range batch {
			grouped[item.userID] = append(grouped[item.userID], item.id)
		}

		for userID, ids := range grouped {
			_ = s.store.DeleteBatchByUser(ctx, userID, ids)
		}

		batch = batch[:0]
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case item := <-s.deleteQueue:
			batch = append(batch, item)
			if len(batch) >= deleteBatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
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
