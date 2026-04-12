package service

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/Fa1ry7a1l/url-shortener/internal/repository"
)

type Shortener struct {
	store          repository.Store
	idgen          IDGenerator
	baseURL        string
	maxSaveRetries int
}

func NewShortener(store repository.Store, idgen IDGenerator, baseURL string) *Shortener {
	baseURL = strings.TrimRight(baseURL, "/")
	return &Shortener{
		store:          store,
		idgen:          idgen,
		baseURL:        baseURL,
		maxSaveRetries: 10,
	}
}

func (s *Shortener) Shorten(ctx context.Context, original string) (string, error) {
	original = strings.TrimSpace(original)
	if !isValidURL(original) {
		return "", errors.New("invalid url")
	}

	for i := 0; i < s.maxSaveRetries; i++ {
		id, err := s.idgen.NewID()
		if err != nil {
			return "", err
		}

		err = s.store.Save(ctx, id, original)
		if err == nil {
			return s.baseURL + "/" + id, nil
		}

		if errors.Is(err, repository.ErrIDExists) {
			continue
		}
		return "", err
	}

	return "", errors.New("failed to generate unique id")
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
	return u.Scheme == "http" || u.Scheme == "https"
}
