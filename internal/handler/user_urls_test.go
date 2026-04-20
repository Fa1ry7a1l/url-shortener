package handler_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Fa1ry7a1l/url-shortener/internal/auth"
	"github.com/Fa1ry7a1l/url-shortener/internal/handler"
	"github.com/Fa1ry7a1l/url-shortener/internal/repository"
	"github.com/Fa1ry7a1l/url-shortener/internal/service"
	"github.com/stretchr/testify/require"
)

func newAuthenticatedTestHandler(t *testing.T) http.Handler {
	t.Helper()

	store := repository.NewMemStore()
	svc := service.NewShortener(store, service.NewRandomID(8), "http://localhost:8080")

	shortenH := handler.NewShortenHandler(svc)
	shortenJSONH := handler.NewShortenJSONHandler(svc)
	shortenBatchH := handler.NewShortenBatchHandler(svc)
	resolveH := handler.NewResolveHandler(svc)
	userURLsH := handler.NewUserURLsHandler(svc)
	deleteUserURLsH := handler.NewDeleteUserURLsHandler(svc)
	pingH := handler.NewPingHandler(store)
	authManager := auth.NewManager("test-secret", auth.DefaultTTL)

	router := handler.NewRouter(
		shortenH.Handle,
		shortenJSONH.Handle,
		shortenBatchH.Handle,
		resolveH.Handle,
		userURLsH.Handle,
		deleteUserURLsH.Handle,
		pingH.Handle,
		nil,
		authManager,
	)

	return router.Handler()
}

func TestAPI_UserURLs_ReturnsUserLinks(t *testing.T) {
	h := newAuthenticatedTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "http://example.com/", bytes.NewBufferString("https://example.com/one"))
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)
	shortURL := rr.Body.String()
	resp := rr.Result()
	defer resp.Body.Close()
	cookies := resp.Cookies()
	require.NotEmpty(t, cookies)

	req = httptest.NewRequest(http.MethodGet, "http://example.com/api/user/urls", nil)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.JSONEq(t, `[
		{
			"short_url": "`+shortURL+`",
			"original_url": "https://example.com/one"
		}
	]`, rr.Body.String())
}

func TestAPI_UserURLs_NoContent(t *testing.T) {
	h := newAuthenticatedTestHandler(t)
	authManager := auth.NewManager("test-secret", auth.DefaultTTL)
	token, err := authManager.NewToken("user-without-links")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/user/urls", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: token})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusNoContent, rr.Code)
}

func TestAPI_UserURLs_UnauthorizedWhenCookieHasNoUserID(t *testing.T) {
	h := newAuthenticatedTestHandler(t)
	authManager := auth.NewManager("test-secret", auth.DefaultTTL)
	token, err := authManager.NewToken("")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/user/urls", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: token})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
}
