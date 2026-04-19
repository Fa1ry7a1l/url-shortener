package handler

import (
	"encoding/json"
	"net/http"

	"github.com/Fa1ry7a1l/url-shortener/internal/service"
)

type shortenBatchRequestItem struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type shortenBatchResponseItem struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

type ShortenBatchHandler struct {
	svc ShortenerService
}

func NewShortenBatchHandler(svc ShortenerService) *ShortenBatchHandler {
	return &ShortenBatchHandler{svc: svc}
}

func (h *ShortenBatchHandler) Handle(w http.ResponseWriter, r *http.Request) {
	var req []shortenBatchRequestItem
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if len(req) == 0 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	items := make([]service.BatchRequestItem, 0, len(req))
	for _, item := range req {
		items = append(items, service.BatchRequestItem{
			CorrelationID: item.CorrelationID,
			OriginalURL:   item.OriginalURL,
		})
	}

	res, err := h.svc.ShortenBatch(serviceContext(r.Context()), items)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	resp := make([]shortenBatchResponseItem, 0, len(res))
	for _, item := range res {
		resp = append(resp, shortenBatchResponseItem{
			CorrelationID: item.CorrelationID,
			ShortURL:      item.ShortURL,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}
