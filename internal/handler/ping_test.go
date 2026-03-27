package handler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Fa1ry7a1l/url-shortener/internal/handler"
)

type fakePinger struct {
	err error
}

func (f fakePinger) Ping(ctx context.Context) error {
	return f.err
}

func TestPingHandler_OK(t *testing.T) {
	h := handler.NewPingHandler(fakePinger{})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rr := httptest.NewRecorder()

	h.Handle(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestPingHandler_Error(t *testing.T) {
	h := handler.NewPingHandler(fakePinger{err: errors.New("db down")})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rr := httptest.NewRecorder()

	h.Handle(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
}
