package service

import "context"

type userIDContextKey struct{}

// UserIDFromContext returns the service user ID stored in ctx.
func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDContextKey{}).(string)
	return userID, ok && userID != ""
}

// ContextWithUserID stores a service user ID in ctx.
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDContextKey{}, userID)
}
