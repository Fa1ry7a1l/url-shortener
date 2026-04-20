package handler_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Fa1ry7a1l/url-shortener/internal/service"
	"github.com/stretchr/testify/require"

	"github.com/Fa1ry7a1l/url-shortener/internal/handler"
)

type gzipFakeSvc struct {
	shortenFn func(ctx context.Context, original string) (string, error)
	resolveFn func(ctx context.Context, id string) (string, error)
}

func (f gzipFakeSvc) ShortenBatch(ctx context.Context, items []service.BatchRequestItem) ([]service.BatchResponseItem, error) {
	return nil, nil
}

func (f gzipFakeSvc) UserURLs(ctx context.Context) ([]service.UserURL, error) {
	return nil, nil
}

func (f gzipFakeSvc) Shorten(ctx context.Context, original string) (string, error) {
	return f.shortenFn(ctx, original)
}

func (f gzipFakeSvc) Resolve(ctx context.Context, id string) (string, error) {
	return f.resolveFn(ctx, id)
}

type noopPinger struct{}

func (noopPinger) Ping(_ context.Context) error { return nil }
func newGzipTestHandler(t *testing.T, svc handler.ShortenerService) http.Handler {
	t.Helper()

	shortenH := handler.NewShortenHandler(svc)
	shortenJSONH := handler.NewShortenJSONHandler(svc)
	resolveH := handler.NewResolveHandler(svc)
	shortenBatchH := handler.NewShortenBatchHandler(svc)
	userURLsH := handler.NewUserURLsHandler(svc)
	pingH := handler.NewPingHandler(noopPinger{})

	router := handler.NewRouter(
		shortenH.Handle,
		shortenJSONH.Handle,
		shortenBatchH.Handle,
		resolveH.Handle,
		userURLsH.Handle,
		pingH.Handle,
		nil,
		nil,
	)

	return router.Handler()
}

func gzipData(t *testing.T, data []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)

	_, err := gz.Write(data)
	require.NoError(t, err)
	require.NoError(t, gz.Close())

	return buf.Bytes()
}

func ungzipData(t *testing.T, r io.Reader) []byte {
	t.Helper()

	gz, err := gzip.NewReader(r)
	require.NoError(t, err)
	defer gz.Close()

	data, err := io.ReadAll(gz)
	require.NoError(t, err)

	return data
}

func TestGzip_JSONEndpoint_CompressedRequest(t *testing.T) {
	svc := gzipFakeSvc{
		shortenFn: func(ctx context.Context, original string) (string, error) {
			require.Equal(t, "https://practicum.yandex.ru", original)
			return "http://localhost:8080/EwHXdJfB", nil
		},
		resolveFn: func(ctx context.Context, id string) (string, error) {
			return "", errors.New("not used")
		},
	}

	h := newGzipTestHandler(t, svc)

	body := gzipData(t, []byte(`{"url":"https://practicum.yandex.ru"}`))

	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/shorten", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)
	require.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	require.JSONEq(t, `{"result":"http://localhost:8080/EwHXdJfB"}`, rr.Body.String())
}

func TestGzip_JSONEndpoint_CompressedResponse(t *testing.T) {
	svc := gzipFakeSvc{
		shortenFn: func(ctx context.Context, original string) (string, error) {
			require.Equal(t, "https://practicum.yandex.ru", original)
			return "http://localhost:8080/EwHXdJfB", nil
		},
		resolveFn: func(ctx context.Context, id string) (string, error) {
			return "", errors.New("not used")
		},
	}

	h := newGzipTestHandler(t, svc)

	req := httptest.NewRequest(
		http.MethodPost,
		"http://example.com/api/shorten",
		bytes.NewBufferString(`{"url":"https://practicum.yandex.ru"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)
	require.Equal(t, "gzip", rr.Header().Get("Content-Encoding"))

	body := ungzipData(t, rr.Body)
	require.JSONEq(t, `{"result":"http://localhost:8080/EwHXdJfB"}`, string(body))
}

func TestGzip_JSONEndpoint_CompressedRequestAndResponse(t *testing.T) {
	svc := gzipFakeSvc{
		shortenFn: func(ctx context.Context, original string) (string, error) {
			require.Equal(t, "https://practicum.yandex.ru", original)
			return "http://localhost:8080/EwHXdJfB", nil
		},
		resolveFn: func(ctx context.Context, id string) (string, error) {
			return "", errors.New("not used")
		},
	}

	h := newGzipTestHandler(t, svc)

	body := gzipData(t, []byte(`{"url":"https://practicum.yandex.ru"}`))

	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/shorten", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)
	require.Equal(t, "gzip", rr.Header().Get("Content-Encoding"))

	respBody := ungzipData(t, rr.Body)
	require.JSONEq(t, `{"result":"http://localhost:8080/EwHXdJfB"}`, string(respBody))
}

func TestGzip_JSONEndpoint_InvalidCompressedRequest(t *testing.T) {
	svc := gzipFakeSvc{
		shortenFn: func(ctx context.Context, original string) (string, error) {
			return "http://localhost:8080/EwHXdJfB", nil
		},
		resolveFn: func(ctx context.Context, id string) (string, error) {
			return "", errors.New("not used")
		},
	}

	h := newGzipTestHandler(t, svc)

	req := httptest.NewRequest(
		http.MethodPost,
		"http://example.com/api/shorten",
		bytes.NewBufferString("not-a-valid-gzip-stream"),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestGzip_PlainTextEndpoint_ResponseNotCompressed(t *testing.T) {
	svc := gzipFakeSvc{
		shortenFn: func(ctx context.Context, original string) (string, error) {
			require.Equal(t, "https://practicum.yandex.ru/", original)
			return "http://localhost:8080/EwHXdJfB", nil
		},
		resolveFn: func(ctx context.Context, id string) (string, error) {
			return "", errors.New("not used")
		},
	}

	h := newGzipTestHandler(t, svc)

	req := httptest.NewRequest(
		http.MethodPost,
		"http://example.com/",
		bytes.NewBufferString("https://practicum.yandex.ru/"),
	)
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Accept-Encoding", "gzip")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)
	require.Empty(t, rr.Header().Get("Content-Encoding"))
	require.Equal(t, "http://localhost:8080/EwHXdJfB", rr.Body.String())
}
