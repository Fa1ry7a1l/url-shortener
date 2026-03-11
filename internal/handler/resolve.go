package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Fa1ry7a1l/url-shortener/internal/repository"
	"github.com/Fa1ry7a1l/url-shortener/internal/service"
)

type ResolveHandler struct {
	svc *service.Shortener
}

func NewResolveHandler(svc *service.Shortener) *ResolveHandler {
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
		if errors.Is(err, repository.ErrNotFound) {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	w.Header().Set("Location", original)
	w.WriteHeader(http.StatusTemporaryRedirect) // 307
}
