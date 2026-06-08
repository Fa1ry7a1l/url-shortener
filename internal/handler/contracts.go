package handler

import (
	"context"

	"github.com/Fa1ry7a1l/url-shortener/internal/audit"
	"github.com/Fa1ry7a1l/url-shortener/internal/service"
)

type ShortenerService interface {
	Shorten(ctx context.Context, original string) (string, error)
	ShortenBatch(ctx context.Context, items []service.BatchRequestItem) ([]service.BatchResponseItem, error)
	Resolve(ctx context.Context, id string) (string, error)
	UserURLs(ctx context.Context) ([]service.UserURL, error)
	DeleteURLs(ctx context.Context, ids []string) error
}

type AuditPublisher interface {
	Notify(ctx context.Context, event audit.Event) error
}
