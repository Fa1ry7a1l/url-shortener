package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Fa1ry7a1l/url-shortener/internal/service"
)

type DeleteUserURLsHandler struct {
	svc ShortenerService
}

func NewDeleteUserURLsHandler(svc ShortenerService) *DeleteUserURLsHandler {
	return &DeleteUserURLsHandler{svc: svc}
}

func (h *DeleteUserURLsHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if ct := r.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(ct, "application/json") {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var ids []string
	if err := json.NewDecoder(r.Body).Decode(&ids); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if err := h.svc.DeleteURLs(serviceContext(r.Context()), ids); err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
