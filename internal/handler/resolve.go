package handler

import (
	"errors"
	"net/http"

	"github.com/Fa1ry7a1l/url-shortener/internal/repository"
	"github.com/go-chi/chi/v5"
)

// ResolveHandler handles redirects from a short ID to the original URL.
type ResolveHandler struct {
	svc     ShortenerService
	auditor AuditPublisher
}

// NewResolveHandler creates a redirect handler.
func NewResolveHandler(svc ShortenerService, auditors ...AuditPublisher) *ResolveHandler {
	h := &ResolveHandler{svc: svc}
	if len(auditors) > 0 {
		h.auditor = auditors[0]
	}
	return h
}

// Handle resolves the {id} route parameter and responds with a temporary redirect.
func (h *ResolveHandler) Handle(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	original, err := h.svc.Resolve(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrDeleted) {
			http.Error(w, "gone", http.StatusGone)
			return
		}
		if errors.Is(err, repository.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", original)
	w.WriteHeader(http.StatusTemporaryRedirect) // 307
	publishFollowAudit(r.Context(), h.auditor, original)
}
