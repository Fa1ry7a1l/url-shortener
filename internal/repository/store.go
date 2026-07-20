package repository

import (
	"context"
	"errors"
)

var (
	// ErrNotFound is returned when a short ID does not exist.
	ErrNotFound = errors.New("not found")
	// ErrDeleted is returned when a short ID was deleted by its owner.
	ErrDeleted = errors.New("deleted")
	// ErrIDExists is returned when a generated short ID already exists.
	ErrIDExists = errors.New("id already exists")
	// ErrOriginalURLExist is returned when the original URL is already stored.
	ErrOriginalURLExist = errors.New("original url already exists")
)

// BatchItem describes one record saved by Store.SaveBatch.
type BatchItem struct {
	// ID is the short URL identifier.
	ID string
	// Original is the original URL.
	Original string
	// UserID owns the short URL.
	UserID string
}

// UserURL describes a stored URL returned for a specific user.
type UserURL struct {
	// ID is the short URL identifier.
	ID string
	// Original is the original URL.
	Original string
}

// Stats contains service-wide storage counters.
type Stats struct {
	// URLs is the number of stored short URLs.
	URLs int
	// Users is the number of users that have stored URLs.
	Users int
}

// Store defines persistence operations required by the shortener service.
type Store interface {
	// Save stores an ID and original URL without binding it to a user.
	Save(ctx context.Context, id string, original string) error
	// SaveForUser stores an ID and original URL for a user.
	SaveForUser(ctx context.Context, id string, original string, userID string) error
	// SaveBatch stores multiple URLs atomically when the implementation supports it.
	SaveBatch(ctx context.Context, items []BatchItem) error
	// DeleteBatchByUser marks user-owned IDs as deleted.
	DeleteBatchByUser(ctx context.Context, userID string, ids []string) error
	// Get returns the original URL by short ID.
	Get(ctx context.Context, id string) (string, error)
	// GetByOriginal returns the short ID previously stored for an original URL.
	GetByOriginal(ctx context.Context, original string) (string, error)
	// GetByUser returns non-deleted URLs created by a user.
	GetByUser(ctx context.Context, userID string) ([]UserURL, error)
	// Stats returns service-wide storage counters.
	Stats(ctx context.Context) (Stats, error)
	// Ping checks whether the storage backend is available.
	Ping(ctx context.Context) error
}
