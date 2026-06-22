package handler

import (
	"context"
	"net/http"
)

// Pinger checks whether the backing storage is available.
type Pinger interface {
	// Ping verifies connectivity to the dependency.
	Ping(ctx context.Context) error
}

// PingHandler handles storage health checks on GET /ping.
type PingHandler struct {
	pinger Pinger
}

// NewPingHandler creates a health-check handler.
func NewPingHandler(pinger Pinger) *PingHandler {
	return &PingHandler{pinger: pinger}
}

// Handle returns 200 when the storage dependency is available.
func (h *PingHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if h.pinger == nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if err := h.pinger.Ping(r.Context()); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
