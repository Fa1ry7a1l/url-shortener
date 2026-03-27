package handler_test

import (
	"bytes"
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

type fakeSvc struct {
	shortenFn func(ctx context.Context, original string) (string, error)
	resolveFn func(ctx context.Context, id string) (string, error)
}

func (f fakeSvc) ShortenBatch(ctx context.Context, items []service.BatchRequestItem) ([]service.BatchResponseItem, error) {
	return nil, nil
}

func (f fakeSvc) Shorten(ctx context.Context, original string) (string, error) {
	return f.shortenFn(ctx, original)
}

func (f fakeSvc) Resolve(ctx context.Context, id string) (string, error) {
	return f.resolveFn(ctx, id)
}

func newTestHandler(t *testing.T, svc handler.ShortenerService) http.Handler {
	t.Helper()

	shortenH := handler.NewShortenHandler(svc)
	shortenJSONH := handler.NewShortenJSONHandler(svc)
	resolveH := handler.NewResolveHandler(svc)
	shortenBatchH := handler.NewShortenBatchHandler(svc)
	pingH := handler.NewPingHandler(noopPinger{})
	router := handler.NewRouter(shortenH.Handle, shortenJSONH.Handle, shortenBatchH.Handle, resolveH.Handle, pingH.Handle, nil)

	return router.Handler()
}

func TestAPI_Shorten_POSTRoot(t *testing.T) {
	const baseURL = "http://localhost:8080"

	type tc struct {
		name        string
		contentType string
		body        string
		svcErr      error
		wantCode    int
		wantBody    string
	}

	tests := []tc{
		{
			name:        "ok_text_plain",
			contentType: "text/plain",
			body:        "https://practicum.yandex.ru/",
			wantCode:    http.StatusCreated,
			wantBody:    baseURL + "/EwHXdJfB",
		},
		{
			name:        "ok_with_charset_and_spaces",
			contentType: "text/plain; charset=utf-8",
			body:        "   https://practicum.yandex.ru/   \n",
			wantCode:    http.StatusCreated,
			wantBody:    baseURL + "/EwHXdJfB",
		},
		{
			name:        "bad_content_type",
			contentType: "application/json",
			body:        "https://practicum.yandex.ru/",
			wantCode:    http.StatusBadRequest,
		},
		{
			name:        "bad_empty_body",
			contentType: "text/plain",
			body:        "",
			wantCode:    http.StatusBadRequest,
		},
		{
			name:        "bad_service_error",
			contentType: "text/plain",
			body:        "https://practicum.yandex.ru/",
			svcErr:      errors.New("boom"),
			wantCode:    http.StatusBadRequest,
		},
		{
			name:        "conflict_existing_url",
			contentType: "text/plain",
			body:        "https://practicum.yandex.ru/",
			svcErr:      &service.ConflictError{ShortURL: "http://localhost:8080/existing"},
			wantCode:    http.StatusConflict,
			wantBody:    "http://localhost:8080/existing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := fakeSvc{
				shortenFn: func(_ context.Context, _ string) (string, error) {
					if tt.svcErr != nil {
						return "", tt.svcErr
					}
					return baseURL + "/EwHXdJfB", nil
				},
				resolveFn: func(_ context.Context, _ string) (string, error) {
					return "", errors.New("not used")
				},
			}

			h := newTestHandler(t, svc)

			req := httptest.NewRequest(http.MethodPost, "http://example.com/", bytes.NewBufferString(tt.body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			rr := httptest.NewRecorder()

			h.ServeHTTP(rr, req)

			require.Equal(t, tt.wantCode, rr.Code)

			if tt.wantCode == http.StatusCreated {
				require.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
				got, _ := io.ReadAll(rr.Body)
				require.Equal(t, tt.wantBody, string(got))
			}
		})
	}
}

func TestAPI_Resolve_GETID(t *testing.T) {
	type tc struct {
		name       string
		path       string
		svcResult  string
		svcErr     error
		wantCode   int
		wantLoc    string
		wantHasLoc bool
	}

	tests := []tc{
		{
			name:       "ok_redirect",
			path:       "/EwHXdJfB",
			svcResult:  "https://practicum.yandex.ru/",
			wantCode:   http.StatusTemporaryRedirect,
			wantLoc:    "https://practicum.yandex.ru/",
			wantHasLoc: true,
		},
		{
			name:       "bad_not_found_or_any_error_becomes_400",
			path:       "/NoSuchID",
			svcErr:     errors.New("not found"),
			wantCode:   http.StatusInternalServerError,
			wantHasLoc: false,
		},
		{
			name:       "bad_multi_segment_path",
			path:       "/a/b",
			wantCode:   http.StatusBadRequest,
			wantHasLoc: false,
		},
		{
			name:       "bad_root_is_not_get_endpoint",
			path:       "/",
			wantCode:   http.StatusBadRequest,
			wantHasLoc: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := fakeSvc{
				shortenFn: func(_ context.Context, _ string) (string, error) {
					return "", errors.New("not used")
				},
				resolveFn: func(_ context.Context, _ string) (string, error) {
					if tt.svcErr != nil {
						return "", tt.svcErr
					}
					return tt.svcResult, nil
				},
			}

			h := newTestHandler(t, svc)

			req := httptest.NewRequest(http.MethodGet, "http://example.com"+tt.path, nil)
			rr := httptest.NewRecorder()

			h.ServeHTTP(rr, req)

			require.Equal(t, tt.wantCode, rr.Code)

			loc := rr.Header().Get("Location")
			if tt.wantHasLoc {
				require.Equal(t, tt.wantLoc, loc)
			} else {
				require.Empty(t, loc)
			}
		})
	}
}

func TestAPI_BadRequests_Return400(t *testing.T) {
	svc := fakeSvc{
		shortenFn: func(_ context.Context, _ string) (string, error) {
			return "http://localhost:8080/ID", nil
		},
		resolveFn: func(_ context.Context, _ string) (string, error) {
			return "https://example.com", nil
		},
	}

	h := newTestHandler(t, svc)

	tests := []struct {
		name   string
		method string
		path   string
		body   string
		ctype  string
	}{
		{"put_not_allowed", http.MethodPut, "/", "https://practicum.yandex.ru/", "text/plain"},
		{"post_not_root", http.MethodPost, "/abc", "https://practicum.yandex.ru/", "text/plain"},
		{"get_multi_segment", http.MethodGet, "/a/b", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body io.Reader
			if tt.body != "" {
				body = bytes.NewBufferString(tt.body)
			}
			req := httptest.NewRequest(tt.method, "http://example.com"+tt.path, body)
			if tt.ctype != "" {
				req.Header.Set("Content-Type", tt.ctype)
			}

			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)

			require.Equal(t, http.StatusBadRequest, rr.Code)
		})
	}
}

func TestAPI_ShortenJSON_POSTAPIShorten(t *testing.T) {
	const baseURL = "http://localhost:8080"

	type tc struct {
		name        string
		contentType string
		body        string
		svcErr      error
		wantCode    int
		wantBody    string
	}

	tests := []tc{
		{
			name:        "ok_json",
			contentType: "application/json",
			body:        `{"url":"https://practicum.yandex.ru"}`,
			wantCode:    http.StatusCreated,
			wantBody:    `{"result":"http://localhost:8080/EwHXdJfB"}` + "\n",
		},
		{
			name:        "ok_json_with_spaces",
			contentType: "application/json",
			body:        "{\n  \"url\": \"https://practicum.yandex.ru\"\n}",
			wantCode:    http.StatusCreated,
			wantBody:    `{"result":"http://localhost:8080/EwHXdJfB"}` + "\n",
		},
		{
			name:        "bad_invalid_json",
			contentType: "application/json",
			body:        `{"url":`,
			wantCode:    http.StatusBadRequest,
		},
		{
			name:        "bad_service_error",
			contentType: "application/json",
			body:        `{"url":"https://practicum.yandex.ru"}`,
			svcErr:      errors.New("boom"),
			wantCode:    http.StatusBadRequest,
		},
		{
			name:        "conflict_existing_url",
			contentType: "application/json",
			body:        `{"url":"https://practicum.yandex.ru"}`,
			svcErr:      &service.ConflictError{ShortURL: "http://localhost:8080/existing"},
			wantCode:    http.StatusConflict,
			wantBody:    `{"result":"http://localhost:8080/existing"}` + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := fakeSvc{
				shortenFn: func(ctx context.Context, original string) (string, error) {
					_ = ctx
					if tt.svcErr != nil {
						return "", tt.svcErr
					}
					require.Equal(t, "https://practicum.yandex.ru", original)
					return baseURL + "/EwHXdJfB", nil
				},
				resolveFn: func(ctx context.Context, id string) (string, error) {
					_ = ctx
					_ = id
					return "", errors.New("not used")
				},
			}

			h := newTestHandler(t, svc)

			req := httptest.NewRequest(
				http.MethodPost,
				"http://example.com/api/shorten",
				bytes.NewBufferString(tt.body),
			)
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)

			require.Equal(t, tt.wantCode, rr.Code)

			if tt.wantCode == http.StatusCreated {
				require.Equal(t, "application/json", rr.Header().Get("Content-Type"))
				got, _ := io.ReadAll(rr.Body)
				require.Equal(t, tt.wantBody, string(got))
			}
		})
	}
}
