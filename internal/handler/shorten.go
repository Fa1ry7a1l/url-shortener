package handler

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/Fa1ry7a1l/url-shortener/internal/service"
)

// ShortenHandler handles plain-text URL shortening requests on POST /.
type ShortenHandler struct {
	svc     ShortenerService
	auditor AuditPublisher
}

// NewShortenHandler creates a plain-text shortening handler.
func NewShortenHandler(svc ShortenerService, auditors ...AuditPublisher) *ShortenHandler {
	h := &ShortenHandler{svc: svc}
	if len(auditors) > 0 {
		h.auditor = auditors[0]
	}
	return h
}

// Handle reads an original URL from the request body and writes the short URL.
func (h *ShortenHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if ct := r.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(ct, "text/plain") {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	body, err := readBodyLimit(r, 4096)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	short, err := h.svc.Shorten(serviceContext(r.Context()), body)
	if err != nil {
		var conflictErr *service.ConflictError
		if errors.As(err, &conflictErr) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusConflict)
			_, _ = io.WriteString(w, conflictErr.ShortURL)
			return
		}

		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_, _ = io.WriteString(w, short)
	publishShortenAudit(r.Context(), h.auditor, body)
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
