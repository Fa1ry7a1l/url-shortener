package handler

import (
	"errors"
	"net/http"
	"strings"

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
	id := strings.TrimPrefix(r.URL.Path, "/")
	if id == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	original, err := h.svc.Resolve(r.Context(), id)
	if err != nil {
		// по ТЗ "любой некорректный запрос" -> 400
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
