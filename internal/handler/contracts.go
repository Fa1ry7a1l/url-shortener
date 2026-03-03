package handler

import "context"

type ShortenerService interface {
	Shorten(ctx context.Context, original string) (string, error)
	Resolve(ctx context.Context, id string) (string, error)
}
