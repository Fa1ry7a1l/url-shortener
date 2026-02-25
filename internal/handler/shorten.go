package handler

import (
	"io"
	"net/http"
	"strings"

	"github.com/Fa1ry7a1l/url-shortener/internal/service"
)

type ShortenHandler struct {
	svc *service.Shortener
}

func NewShortenHandler(svc *service.Shortener) *ShortenHandler {
	return &ShortenHandler{svc: svc}
}

func (h *ShortenHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// Content-Type: text/plain (в реальности может прилетать с charset)
	if ct := r.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(ct, "text/plain") {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	body, err := readBodyLimit(r, 4096)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	short, err := h.svc.Shorten(r.Context(), body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_, _ = io.WriteString(w, short)
}

func readBodyLimit(r *http.Request, maxBytes int64) (string, error) {
	if r.Body == nil {
		return "", io.EOF
	}
	defer r.Body.Close()

	lr := io.LimitReader(r.Body, maxBytes+1)
	b, err := io.ReadAll(lr)
	if err != nil {
		return "", err
	}
	if int64(len(b)) > maxBytes || len(b) == 0 {
		return "", io.EOF
	}
	return string(b), nil
}
