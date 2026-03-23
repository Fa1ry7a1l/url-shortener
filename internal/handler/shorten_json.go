package handler

import (
	"encoding/json"
	"net/http"
)

type shortenJSONRequest struct {
	URL string `json:"url"`
}

type shortenJSONResponse struct {
	Result string `json:"result"`
}

type ShortenJSONHandler struct {
	svc ShortenerService
}

func NewShortenJSONHandler(svc ShortenerService) *ShortenJSONHandler {
	return &ShortenJSONHandler{svc: svc}
}

func (h *ShortenJSONHandler) Handle(w http.ResponseWriter, r *http.Request) {
	var req shortenJSONRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	short, err := h.svc.Shorten(r.Context(), req.URL)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	resp := shortenJSONResponse{
		Result: short,
	}
	_ = json.NewEncoder(w).Encode(resp)
}
