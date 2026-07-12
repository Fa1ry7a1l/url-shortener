package handler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Fa1ry7a1l/url-shortener/internal/handler"
	"github.com/Fa1ry7a1l/url-shortener/internal/service"
)

type fakeStatsService struct {
	stats service.Stats
	err   error
}

func (f fakeStatsService) Stats(ctx context.Context) (service.Stats, error) {
	return f.stats, f.err
}

func TestStatsHandler_OK(t *testing.T) {
	h := handler.NewStatsHandler(
		fakeStatsService{stats: service.Stats{URLs: 10, Users: 3}},
		"192.168.1.0/24",
	)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
	req.Header.Set("X-Real-IP", "192.168.1.42")
	rr := httptest.NewRecorder()

	h.Handle(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	require.JSONEq(t, `{"urls":10,"users":3}`, rr.Body.String())
}

func TestStatsHandler_Forbidden(t *testing.T) {
	tests := []struct {
		name          string
		trustedSubnet string
		realIP        string
	}{
		{
			name:          "empty_trusted_subnet",
			trustedSubnet: "",
			realIP:        "192.168.1.42",
		},
		{
			name:          "missing_real_ip",
			trustedSubnet: "192.168.1.0/24",
		},
		{
			name:          "invalid_real_ip",
			trustedSubnet: "192.168.1.0/24",
			realIP:        "not-an-ip",
		},
		{
			name:          "outside_trusted_subnet",
			trustedSubnet: "192.168.1.0/24",
			realIP:        "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := handler.NewStatsHandler(
				fakeStatsService{stats: service.Stats{URLs: 10, Users: 3}},
				tt.trustedSubnet,
			)

			req := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
			if tt.realIP != "" {
				req.Header.Set("X-Real-IP", tt.realIP)
			}
			rr := httptest.NewRecorder()

			h.Handle(rr, req)

			require.Equal(t, http.StatusForbidden, rr.Code)
		})
	}
}

func TestStatsHandler_ServiceError(t *testing.T) {
	h := handler.NewStatsHandler(
		fakeStatsService{err: errors.New("stats failed")},
		"192.168.1.0/24",
	)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
	req.Header.Set("X-Real-IP", "192.168.1.42")
	rr := httptest.NewRecorder()

	h.Handle(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestRouter_StatsEndpoint(t *testing.T) {
	statsH := handler.NewStatsHandler(
		fakeStatsService{stats: service.Stats{URLs: 2, Users: 1}},
		"10.0.0.0/8",
	)
	unused := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not used", http.StatusInternalServerError)
	})
	router := handler.NewRouter(
		unused,
		unused,
		unused,
		unused,
		unused,
		unused,
		unused,
		nil,
		nil,
		handler.WithStatsHandler(statsH.Handle),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
	req.Header.Set("X-Real-IP", "10.1.2.3")
	rr := httptest.NewRecorder()

	router.Handler().ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.JSONEq(t, `{"urls":2,"users":1}`, rr.Body.String())
}
