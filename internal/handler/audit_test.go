package handler_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Fa1ry7a1l/url-shortener/internal/audit"
	"github.com/Fa1ry7a1l/url-shortener/internal/auth"
	"github.com/Fa1ry7a1l/url-shortener/internal/handler"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

type auditRecorder struct {
	events []audit.Event
}

func (r *auditRecorder) Notify(_ context.Context, event audit.Event) error {
	r.events = append(r.events, event)
	return nil
}

func TestAudit_PostRoot_Success(t *testing.T) {
	recorder := &auditRecorder{}
	svc := fakeSvc{
		shortenFn: func(_ context.Context, original string) (string, error) {
			require.Equal(t, "  https://example.com/one  \n", original)
			return "http://localhost:8080/abc", nil
		},
		resolveFn: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("not used")
		},
	}
	h := handler.NewShortenHandler(svc, recorder)

	req := httptest.NewRequest(http.MethodPost, "http://example.com/", bytes.NewBufferString("  https://example.com/one  \n"))
	req.Header.Set("Content-Type", "text/plain")
	req = req.WithContext(auth.ContextWithUserID(req.Context(), "user-1"))
	rr := httptest.NewRecorder()

	h.Handle(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)
	require.Len(t, recorder.events, 1)
	require.Equal(t, audit.ActionShorten, recorder.events[0].Action)
	require.Equal(t, "user-1", recorder.events[0].UserID)
	require.Equal(t, "https://example.com/one", recorder.events[0].URL)
	require.NotZero(t, recorder.events[0].TS)
}

func TestAudit_PostAPIShorten_Success(t *testing.T) {
	recorder := &auditRecorder{}
	svc := fakeSvc{
		shortenFn: func(_ context.Context, original string) (string, error) {
			require.Equal(t, " https://example.com/json ", original)
			return "http://localhost:8080/json", nil
		},
		resolveFn: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("not used")
		},
	}
	h := handler.NewShortenJSONHandler(svc, recorder)

	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/shorten", bytes.NewBufferString(`{"url":" https://example.com/json "}`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.ContextWithUserID(req.Context(), "user-1"))
	rr := httptest.NewRecorder()

	h.Handle(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)
	require.Len(t, recorder.events, 1)
	require.Equal(t, audit.ActionShorten, recorder.events[0].Action)
	require.Equal(t, "user-1", recorder.events[0].UserID)
	require.Equal(t, "https://example.com/json", recorder.events[0].URL)
	require.NotZero(t, recorder.events[0].TS)
}

func TestAudit_GetID_Success(t *testing.T) {
	recorder := &auditRecorder{}
	svc := fakeSvc{
		shortenFn: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("not used")
		},
		resolveFn: func(_ context.Context, id string) (string, error) {
			require.Equal(t, "abc", id)
			return "https://example.com/follow", nil
		},
	}
	h := handler.NewResolveHandler(svc, recorder)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/abc", nil)
	req = req.WithContext(auth.ContextWithUserID(req.Context(), "user-1"))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", "abc")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
	rr := httptest.NewRecorder()

	h.Handle(rr, req)

	require.Equal(t, http.StatusTemporaryRedirect, rr.Code)
	require.Len(t, recorder.events, 1)
	require.Equal(t, audit.ActionFollow, recorder.events[0].Action)
	require.Equal(t, "user-1", recorder.events[0].UserID)
	require.Equal(t, "https://example.com/follow", recorder.events[0].URL)
	require.NotZero(t, recorder.events[0].TS)
}
