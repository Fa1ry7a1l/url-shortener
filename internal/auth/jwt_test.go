package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestManager_NewTokenAndParse(t *testing.T) {
	manager := NewManager("test-secret", DefaultTTL)

	token, err := manager.NewToken("user-1")
	require.NoError(t, err)

	claims, err := manager.Parse(token)
	require.NoError(t, err)
	require.Equal(t, "user-1", claims.UserID)
}

func TestManager_ParseRejectsTamperedToken(t *testing.T) {
	manager := NewManager("test-secret", DefaultTTL)

	token, err := manager.NewToken("user-1")
	require.NoError(t, err)

	tampered := strings.TrimSuffix(token, token[len(token)-1:]) + "x"
	_, err = manager.Parse(tampered)
	require.Error(t, err)
}

func TestManager_MiddlewareIssuesCookieWhenMissing(t *testing.T) {
	manager := NewManager("test-secret", DefaultTTL)
	var gotUserID string

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID, _ = UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	manager.Middleware(next).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.NotEmpty(t, gotUserID)

	cookie := findAuthCookie(rr.Result().Cookies())
	require.NotNil(t, cookie)

	claims, err := manager.Parse(cookie.Value)
	require.NoError(t, err)
	require.Equal(t, gotUserID, claims.UserID)
}

func TestManager_MiddlewareReissuesCookieWhenInvalid(t *testing.T) {
	manager := NewManager("test-secret", DefaultTTL)
	var gotUserID string

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID, _ = UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: "invalid-token"})
	rr := httptest.NewRecorder()
	manager.Middleware(next).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.NotEmpty(t, gotUserID)
	require.NotNil(t, findAuthCookie(rr.Result().Cookies()))
}

func TestManager_MiddlewareDoesNotAuthenticateTokenWithoutUserID(t *testing.T) {
	manager := NewManager("test-secret", DefaultTTL)
	token, err := manager.NewToken("")
	require.NoError(t, err)
	var authenticated bool

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, authenticated = UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: token})
	rr := httptest.NewRecorder()
	manager.Middleware(next).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.False(t, authenticated)
	require.Nil(t, findAuthCookie(rr.Result().Cookies()))
}

func findAuthCookie(cookies []*http.Cookie) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == CookieName {
			return cookie
		}
	}
	return nil
}
