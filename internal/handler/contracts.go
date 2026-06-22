package handler

import (
	"context"

	"github.com/Fa1ry7a1l/url-shortener/internal/audit"
	"github.com/Fa1ry7a1l/url-shortener/internal/service"
)

// ShortenerService defines the URL shortener operations required by HTTP handlers.
type ShortenerService interface {
	// Shorten validates and stores one original URL.
	Shorten(ctx context.Context, original string) (string, error)
	// ShortenBatch validates and stores a batch of original URLs.
	ShortenBatch(ctx context.Context, items []service.BatchRequestItem) ([]service.BatchResponseItem, error)
	// Resolve returns the original URL for a short ID.
	Resolve(ctx context.Context, id string) (string, error)
	// UserURLs returns links created by the authenticated user.
	UserURLs(ctx context.Context) ([]service.UserURL, error)
	// DeleteURLs schedules links created by the authenticated user for deletion.
	DeleteURLs(ctx context.Context, ids []string) error
}

// AuditPublisher receives audit events produced by HTTP handlers.
type AuditPublisher interface {
	// Notify publishes one audit event.
	Notify(ctx context.Context, event audit.Event) error
}
