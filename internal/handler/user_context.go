package handler

import (
	"context"

	"github.com/Fa1ry7a1l/url-shortener/internal/auth"
	"github.com/Fa1ry7a1l/url-shortener/internal/service"
)

func serviceContext(ctx context.Context) context.Context {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return ctx
	}
	return service.ContextWithUserID(ctx, userID)
}
