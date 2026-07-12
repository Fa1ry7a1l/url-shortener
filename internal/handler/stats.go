package handler

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
)

const realIPHeader = "X-Real-IP"

type statsResponse struct {
	URLs  int `json:"urls"`
	Users int `json:"users"`
}

// StatsHandler handles service statistics for trusted internal clients.
type StatsHandler struct {
	svc           StatsService
	trustedSubnet *net.IPNet
}

// NewStatsHandler creates an internal statistics handler.
func NewStatsHandler(svc StatsService, trustedSubnet string) *StatsHandler {
	var ipNet *net.IPNet
	if trustedSubnet != "" {
		_, parsedNet, err := net.ParseCIDR(trustedSubnet)
		if err == nil {
			ipNet = parsedNet
		}
	}

	return &StatsHandler{
		svc:           svc,
		trustedSubnet: ipNet,
	}
}

// Handle writes service-wide counters for requests from the trusted subnet.
func (h *StatsHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if h.trustedSubnet == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	clientIP := net.ParseIP(strings.TrimSpace(r.Header.Get(realIPHeader)))
	if clientIP == nil || !h.trustedSubnet.Contains(clientIP) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if h.svc == nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	stats, err := h.svc.Stats(r.Context())
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(statsResponse{
		URLs:  stats.URLs,
		Users: stats.Users,
	})
}
