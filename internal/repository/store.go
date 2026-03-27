package repository

import (
	"context"
	"errors"
)

var (
	ErrNotFound = errors.New("not found")
	ErrIDExists = errors.New("id already exists")
)

type BatchItem struct {
	ID       string
	Original string
}

type Store interface {
	Save(ctx context.Context, id string, original string) error
	SaveBatch(ctx context.Context, items []BatchItem) error
	Get(ctx context.Context, id string) (string, error)
	Ping(ctx context.Context) error
}
