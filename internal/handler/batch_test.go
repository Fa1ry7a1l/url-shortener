package handler_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Fa1ry7a1l/url-shortener/internal/handler"
	"github.com/Fa1ry7a1l/url-shortener/internal/service"
)

type batchFakeSvc struct {
	shortenFn      func(ctx context.Context, original string) (string, error)
	shortenBatchFn func(ctx context.Context, items []service.BatchRequestItem) ([]service.BatchResponseItem, error)
	resolveFn      func(ctx context.Context, id string) (string, error)
}

func (f batchFakeSvc) Shorten(ctx context.Context, original string) (string, error) {
	return f.shortenFn(ctx, original)
}

func (f batchFakeSvc) ShortenBatch(ctx context.Context, items []service.BatchRequestItem) ([]service.BatchResponseItem, error) {
	return f.shortenBatchFn(ctx, items)
}

func (f batchFakeSvc) Resolve(ctx context.Context, id string) (string, error) {
	return f.resolveFn(ctx, id)
}

func newBatchTestHandler(t *testing.T, svc handler.ShortenerService) http.Handler {
	t.Helper()

	shortenH := handler.NewShortenHandler(svc)
	shortenJSONH := handler.NewShortenJSONHandler(svc)
	shortenBatchH := handler.NewShortenBatchHandler(svc)
	resolveH := handler.NewResolveHandler(svc)
	pingH := handler.NewPingHandler(noopPinger{})

	router := handler.NewRouter(
		shortenH.Handle,
		shortenJSONH.Handle,
		shortenBatchH.Handle,
		resolveH.Handle,
		pingH.Handle,
		nil,
	)

	return router.Handler()
}

func TestAPI_ShortenBatch_OK(t *testing.T) {
	svc := batchFakeSvc{
		shortenFn: func(ctx context.Context, original string) (string, error) {
			return "", errors.New("not used")
		},
		shortenBatchFn: func(ctx context.Context, items []service.BatchRequestItem) ([]service.BatchResponseItem, error) {
			require.Len(t, items, 2)
			return []service.BatchResponseItem{
				{CorrelationID: "1", ShortURL: "http://localhost:8080/aaa"},
				{CorrelationID: "2", ShortURL: "http://localhost:8080/bbb"},
			}, nil
		},
		resolveFn: func(ctx context.Context, id string) (string, error) {
			return "", errors.New("not used")
		},
	}

	h := newBatchTestHandler(t, svc)

	body := `[
		{"correlation_id":"1","original_url":"https://example.com/1"},
		{"correlation_id":"2","original_url":"https://example.com/2"}
	]`

	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/shorten/batch", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)
	require.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	require.JSONEq(t, `[
		{"correlation_id":"1","short_url":"http://localhost:8080/aaa"},
		{"correlation_id":"2","short_url":"http://localhost:8080/bbb"}
	]`, rr.Body.String())
}

func TestAPI_ShortenBatch_BadRequest(t *testing.T) {
	svc := batchFakeSvc{
		shortenFn: func(ctx context.Context, original string) (string, error) {
			return "", errors.New("not used")
		},
		shortenBatchFn: func(ctx context.Context, items []service.BatchRequestItem) ([]service.BatchResponseItem, error) {
			return nil, errors.New("bad batch")
		},
		resolveFn: func(ctx context.Context, id string) (string, error) {
			return "", errors.New("not used")
		},
	}

	h := newBatchTestHandler(t, svc)

	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/shorten/batch", bytes.NewBufferString(`[]`))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}
