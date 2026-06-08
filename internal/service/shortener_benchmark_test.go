package service_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Fa1ry7a1l/url-shortener/internal/repository"
	"github.com/Fa1ry7a1l/url-shortener/internal/service"
)

func BenchmarkShortenerUserURLs(b *testing.B) {
	const (
		users       = 4
		urlsPerUser = 1024
		targetUser  = "user-02"
	)

	store := repository.NewMemStore()
	for user := 0; user < users; user++ {
		userID := fmt.Sprintf("user-%02d", user)
		for i := 0; i < urlsPerUser; i++ {
			id := fmt.Sprintf("id-%02d-%05d", user, i)
			original := fmt.Sprintf("https://example.com/users/%02d/articles/%05d?utm_source=benchmark", user, i)
			require.NoError(b, store.SaveForUser(context.Background(), id, original, userID))
		}
	}

	svc := service.NewShortener(store, service.NewRandomID(8), "http://localhost:8080")
	ctx := service.ContextWithUserID(context.Background(), targetUser)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		items, err := svc.UserURLs(ctx)
		if err != nil {
			b.Fatal(err)
		}
		if len(items) != urlsPerUser {
			b.Fatalf("expected %d items, got %d", urlsPerUser, len(items))
		}
	}
}
