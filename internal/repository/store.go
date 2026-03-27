package repository

import (
	"context"
	"errors"
)

var (
	ErrNotFound         = errors.New("not found")
	ErrIDExists         = errors.New("id already exists")
	ErrOriginalURLExist = errors.New("original url already exists")
)

type BatchItem struct {
	ID       string
	Original string
}

type Store interface {
	Save(ctx context.Context, id string, original string) error
	SaveBatch(ctx context.Context, items []BatchItem) error
	Get(ctx context.Context, id string) (string, error)
	GetByOriginal(ctx context.Context, original string) (string, error)
	Ping(ctx context.Context) error
}
