package repository

import (
	"context"
	"errors"
)

var (
	ErrNotFound = errors.New("not found")
	ErrIDExists = errors.New("id already exists")
)

type Store interface {
	Save(ctx context.Context, id string, original string) error
	Get(ctx context.Context, id string) (string, error)
	Ping(ctx context.Context) error
}
