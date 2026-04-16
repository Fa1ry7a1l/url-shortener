package handler

import (
	"errors"
	"net/http"

	"github.com/Fa1ry7a1l/url-shortener/internal/repository"
	"github.com/go-chi/chi/v5"
)

type ResolveHandler struct {
	svc ShortenerService
}

func NewResolveHandler(svc ShortenerService) *ResolveHandler {
	return &ResolveHandler{svc: svc}
}

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
}
