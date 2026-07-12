package grpcserver

import (
	"context"
	"strings"

	"github.com/Fa1ry7a1l/url-shortener/internal/auth"
	"github.com/Fa1ry7a1l/url-shortener/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const authorizationMetadata = "authorization"

// AuthUnaryInterceptor reads auth metadata and stores the user ID in context.
func AuthUnaryInterceptor(manager *auth.Manager) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if manager == nil {
			return handler(ctx, req)
		}

		token, ok := tokenFromMetadata(ctx)
		if !ok {
			return handler(ctx, req)
		}

		claims, err := manager.Parse(token)
		if err != nil || claims.UserID == "" {
			return nil, status.Error(codes.Unauthenticated, "invalid authorization")
		}

		ctx = auth.ContextWithUserID(ctx, claims.UserID)
		ctx = service.ContextWithUserID(ctx, claims.UserID)
		return handler(ctx, req)
	}
}

func tokenFromMetadata(ctx context.Context) (string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false
	}

	values := md.Get(authorizationMetadata)
	if len(values) == 0 {
		return "", false
	}

	token := strings.TrimSpace(values[0])
	if token == "" {
		return "", false
	}

	const bearerPrefix = "bearer "
	if len(token) >= len(bearerPrefix) && strings.EqualFold(token[:len(bearerPrefix)], bearerPrefix) {
		token = strings.TrimSpace(token[len(bearerPrefix):])
	}
	return token, token != ""
}
