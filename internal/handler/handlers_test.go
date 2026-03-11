package handler_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Fa1ry7a1l/url-shortener/internal/handler"
	"github.com/Fa1ry7a1l/url-shortener/internal/repository"
	"github.com/Fa1ry7a1l/url-shortener/internal/service"
)

// deterministic ID generator for tests
type fixedIDGen struct {
	id  string
	err error
}

func (g fixedIDGen) NewID() (string, error) {
	return g.id, g.err
}

func newTestHandler(t *testing.T, baseURL string, gen service.IDGenerator, store repository.URLStore) http.Handler {
	t.Helper()

	svc := service.NewShortener(store, gen, baseURL)
	shortenH := handler.NewShortenHandler(svc)
	resolveH := handler.NewResolveHandler(svc)
	router := handler.NewRouter(shortenH.Handle, resolveH.Handle)

	return router.Handler()
}

func TestAPI_Shorten_POSTRoot(t *testing.T) {
	const baseURL = "http://localhost:8080"

	type tc struct {
		name        string
		contentType string
		body        string
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
			name:        "ok_with_spaces_trimmed",
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
			name:        "bad_invalid_url_no_scheme",
			contentType: "text/plain",
			body:        "practicum.yandex.ru",
			wantCode:    http.StatusBadRequest,
		},
		{
			name:        "bad_invalid_scheme",
			contentType: "text/plain",
			body:        "ftp://example.com/file",
			wantCode:    http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := repository.NewMemStore()
			h := newTestHandler(t, baseURL, fixedIDGen{id: "EwHXdJfB"}, store)

			req := httptest.NewRequest(http.MethodPost, "http://example.com/", bytes.NewBufferString(tt.body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			rr := httptest.NewRecorder()

			h.ServeHTTP(rr, req)

			require.Equal(t, tt.wantCode, rr.Code)

			gotBodyBytes, _ := io.ReadAll(rr.Body)
			gotBody := string(gotBodyBytes)

			if tt.wantCode == http.StatusCreated {
				require.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
				require.Equal(t, tt.wantBody, gotBody)
			} else {
				// На ошибках не проверяем точный текст/формат — только статус (API-устойчивость)
				require.NotEqual(t, http.StatusCreated, rr.Code)
			}
		})
	}
}

func TestAPI_Resolve_GETID(t *testing.T) {
	const baseURL = "http://localhost:8080"

	type tc struct {
		name       string
		path       string
		prepare    func(store repository.URLStore)
		wantCode   int
		wantLoc    string
		wantHasLoc bool
	}

	tests := []tc{
		{
			name: "ok_redirect",
			path: "/EwHXdJfB",
			prepare: func(store repository.URLStore) {
				_ = store.Save(context.Background(), "EwHXdJfB", "https://practicum.yandex.ru/")
			},
			wantCode:   http.StatusTemporaryRedirect,
			wantLoc:    "https://practicum.yandex.ru/",
			wantHasLoc: true,
		},
		{
			name: "bad_not_found_treated_as_bad_request",
			path: "/NoSuchID",
			prepare: func(store repository.URLStore) {
				// nothing
			},
			wantCode:   http.StatusBadRequest,
			wantHasLoc: false,
		},
		{
			name:       "bad_root_is_not_get_endpoint",
			path:       "/",
			prepare:    func(store repository.URLStore) {},
			wantCode:   http.StatusBadRequest,
			wantHasLoc: false,
		},
		{
			name:       "bad_multi_segment_path",
			path:       "/a/b",
			prepare:    func(store repository.URLStore) {},
			wantCode:   http.StatusBadRequest,
			wantHasLoc: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := repository.NewMemStore()
			tt.prepare(store)

			h := newTestHandler(t, baseURL, fixedIDGen{id: "EwHXdJfB"}, store)

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
	const baseURL = "http://localhost:8080"

	tests := []struct {
		name   string
		method string
		path   string
		body   string
		ctype  string
	}{
		{"put_not_allowed", http.MethodPut, "/", "https://practicum.yandex.ru/", "text/plain"},
		{"post_not_root", http.MethodPost, "/abc", "https://practicum.yandex.ru/", "text/plain"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := repository.NewMemStore()
			h := newTestHandler(t, baseURL, fixedIDGen{id: "EwHXdJfB"}, store)

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
