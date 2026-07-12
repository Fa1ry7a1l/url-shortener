package grpcserver_test

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	shortenerpb "github.com/Fa1ry7a1l/url-shortener/api"
	"github.com/Fa1ry7a1l/url-shortener/internal/auth"
	"github.com/Fa1ry7a1l/url-shortener/internal/grpcserver"
	"github.com/Fa1ry7a1l/url-shortener/internal/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
)

type fakeShortener struct {
	shortenFn  func(context.Context, string) (string, error)
	resolveFn  func(context.Context, string) (string, error)
	userURLsFn func(context.Context) ([]service.UserURL, error)
}

func (f fakeShortener) Shorten(ctx context.Context, original string) (string, error) {
	if f.shortenFn != nil {
		return f.shortenFn(ctx, original)
	}
	return "", errors.New("not implemented")
}

func (f fakeShortener) Resolve(ctx context.Context, id string) (string, error) {
	if f.resolveFn != nil {
		return f.resolveFn(ctx, id)
	}
	return "", errors.New("not implemented")
}

func (f fakeShortener) UserURLs(ctx context.Context) ([]service.UserURL, error) {
	if f.userURLsFn != nil {
		return f.userURLsFn(ctx)
	}
	return nil, errors.New("not implemented")
}

func TestServer_MethodsWithAuthorizationMetadata(t *testing.T) {
	authManager := auth.NewManager("test-secret", time.Hour)
	token, err := authManager.NewToken("user-1")
	require.NoError(t, err)

	svc := fakeShortener{
		shortenFn: func(ctx context.Context, original string) (string, error) {
			require.Equal(t, "https://example.com/long", original)
			userID, ok := service.UserIDFromContext(ctx)
			require.True(t, ok)
			require.Equal(t, "user-1", userID)
			return "http://localhost:8080/abc", nil
		},
		resolveFn: func(_ context.Context, id string) (string, error) {
			require.Equal(t, "abc", id)
			return "https://example.com/long", nil
		},
		userURLsFn: func(ctx context.Context) ([]service.UserURL, error) {
			userID, ok := service.UserIDFromContext(ctx)
			require.True(t, ok)
			require.Equal(t, "user-1", userID)
			return []service.UserURL{
				{ShortURL: "http://localhost:8080/abc", OriginalURL: "https://example.com/long"},
			}, nil
		},
	}

	client, cleanup := newTestClient(t, authManager, svc)
	defer cleanup()

	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+token)

	shortenResp, err := client.ShortenURL(ctx, &shortenerpb.URLShortenRequest{Url: "https://example.com/long"})
	require.NoError(t, err)
	require.Equal(t, "http://localhost:8080/abc", shortenResp.GetResult())

	expandResp, err := client.ExpandURL(context.Background(), &shortenerpb.URLExpandRequest{Id: "abc"})
	require.NoError(t, err)
	require.Equal(t, "https://example.com/long", expandResp.GetResult())

	userURLsResp, err := client.ListUserURLs(ctx, &emptypb.Empty{})
	require.NoError(t, err)
	require.Len(t, userURLsResp.GetUrl(), 1)
	require.Equal(t, "http://localhost:8080/abc", userURLsResp.GetUrl()[0].GetShortUrl())
	require.Equal(t, "https://example.com/long", userURLsResp.GetUrl()[0].GetOriginalUrl())
}

func TestServer_ListUserURLsUnauthorized(t *testing.T) {
	authManager := auth.NewManager("test-secret", time.Hour)
	svc := fakeShortener{
		userURLsFn: func(context.Context) ([]service.UserURL, error) {
			return nil, service.ErrUnauthorized
		},
	}

	client, cleanup := newTestClient(t, authManager, svc)
	defer cleanup()

	_, err := client.ListUserURLs(context.Background(), &emptypb.Empty{})
	require.Error(t, err)
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestServer_InvalidAuthorizationMetadata(t *testing.T) {
	authManager := auth.NewManager("test-secret", time.Hour)
	client, cleanup := newTestClient(t, authManager, fakeShortener{})
	defer cleanup()

	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "not-a-token")
	_, err := client.ExpandURL(ctx, &shortenerpb.URLExpandRequest{Id: "abc"})
	require.Error(t, err)
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func newTestClient(t *testing.T, authManager *auth.Manager, svc grpcserver.ShortenerService) (shortenerpb.ShortenerServiceClient, func()) {
	t.Helper()

	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer(grpc.UnaryInterceptor(grpcserver.AuthUnaryInterceptor(authManager)))
	shortenerpb.RegisterShortenerServiceServer(server, grpcserver.NewServer(svc))
	go func() {
		_ = server.Serve(listener)
	}()

	dialer := func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}
	conn, err := grpc.DialContext(
		context.Background(),
		"bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	cleanup := func() {
		require.NoError(t, conn.Close())
		server.Stop()
		_ = listener.Close()
	}
	return shortenerpb.NewShortenerServiceClient(conn), cleanup
}
