package handler

import (
	"context"
	"net/http"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type PingHandler struct {
	pinger Pinger
}

func NewPingHandler(pinger Pinger) *PingHandler {
	return &PingHandler{pinger: pinger}
}

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
