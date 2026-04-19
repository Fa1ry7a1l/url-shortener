package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Fa1ry7a1l/url-shortener/internal/service"
)

type userURLResponseItem struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type UserURLsHandler struct {
	svc ShortenerService
}

func NewUserURLsHandler(svc ShortenerService) *UserURLsHandler {
	return &UserURLsHandler{svc: svc}
}

func (h *UserURLsHandler) Handle(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.UserURLs(serviceContext(r.Context()))
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if len(items) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	resp := make([]userURLResponseItem, 0, len(items))
	for _, item := range items {
		resp = append(resp, userURLResponseItem{
			ShortURL:    item.ShortURL,
			OriginalURL: item.OriginalURL,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
