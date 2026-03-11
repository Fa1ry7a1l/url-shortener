package service

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/Fa1ry7a1l/url-shortener/internal/repository"
)

type Shortener struct {
	store   repository.URLStore
	idgen   IDGenerator
	baseURL string
}

func NewShortener(store repository.URLStore, idgen IDGenerator, baseURL string) *Shortener {
	baseURL = strings.TrimRight(baseURL, "/")
	return &Shortener{store: store, idgen: idgen, baseURL: baseURL}
}

func (s *Shortener) Shorten(ctx context.Context, original string) (string, error) {
	original = strings.TrimSpace(original)
	if !isValidURL(original) {
		return "", errors.New("invalid url")
	}

	id, err := s.idgen.NewID()
	if err != nil {
		return "", err
	}

	if err := s.store.Save(ctx, id, original); err != nil {
		return "", err
	}

	return s.baseURL + "/" + id, nil
}

func (s *Shortener) Resolve(ctx context.Context, id string) (string, error) {
	if id == "" || strings.Contains(id, "/") {
		return "", errors.New("invalid id")
	}
	return s.store.Get(ctx, id)
}

func isValidURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if u.Scheme == "" || u.Host == "" {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	return true
}
