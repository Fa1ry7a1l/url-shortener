package handler

import (
	"context"
	"strings"

	"github.com/Fa1ry7a1l/url-shortener/internal/audit"
)

func publishShortenAudit(ctx context.Context, publisher AuditPublisher, originalURL string) {
	publishAudit(ctx, publisher, audit.ActionShorten, originalURL)
}

func publishFollowAudit(ctx context.Context, publisher AuditPublisher, originalURL string) {
	publishAudit(ctx, publisher, audit.ActionFollow, originalURL)
}

func publishAudit(ctx context.Context, publisher AuditPublisher, action string, originalURL string) {
	if publisher == nil {
		return
	}

	event := audit.NewEvent(action, userIDFromContext(ctx), strings.TrimSpace(originalURL))
	_ = publisher.Notify(ctx, event)
}
