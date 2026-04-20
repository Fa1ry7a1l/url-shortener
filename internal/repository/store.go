package repository

import (
	"context"
	"errors"
)

var (
	ErrNotFound         = errors.New("not found")
	ErrDeleted          = errors.New("deleted")
	ErrIDExists         = errors.New("id already exists")
	ErrOriginalURLExist = errors.New("original url already exists")
)

type BatchItem struct {
	ID       string
	Original string
	UserID   string
}

type UserURL struct {
	ID       string
	Original string
}

type Store interface {
	Save(ctx context.Context, id string, original string) error
	SaveForUser(ctx context.Context, id string, original string, userID string) error
	SaveBatch(ctx context.Context, items []BatchItem) error
	DeleteBatchByUser(ctx context.Context, userID string, ids []string) error
	Get(ctx context.Context, id string) (string, error)
	GetByOriginal(ctx context.Context, original string) (string, error)
	GetByUser(ctx context.Context, userID string) ([]UserURL, error)
	Ping(ctx context.Context) error
}
