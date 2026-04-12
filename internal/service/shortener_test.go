package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Fa1ry7a1l/url-shortener/internal/repository"
	"github.com/Fa1ry7a1l/url-shortener/internal/service"
)

type fixedIDGen struct {
	id  string
	err error
}

func (g fixedIDGen) NewID() (string, error) { return g.id, g.err }

type errStore struct {
	err error
}

func (e errStore) GetByOriginal(ctx context.Context, original string) (string, error) {
	return "", e.err
}

func (e errStore) Save(ctx context.Context, id string, original string) error {
	return e.err
}

func (e errStore) SaveBatch(ctx context.Context, items []repository.BatchItem) error {
	return e.err
}

func (e errStore) Get(ctx context.Context, id string) (string, error) {
	return "", e.err
}

func (e errStore) Ping(ctx context.Context) error {
	return e.err
}

func TestShortener_Shorten_OK(t *testing.T) {
	store := repository.NewMemStore()
	gen := fixedIDGen{id: "EwHXdJfB"}

	svc := service.NewShortener(store, gen, "http://localhost:8080")

	short, err := svc.Shorten(context.Background(), "https://practicum.yandex.ru/")
	require.NoError(t, err)
	require.Equal(t, "http://localhost:8080/EwHXdJfB", short)

	// ensure saved
	got, err := store.Get(context.Background(), "EwHXdJfB")
	require.NoError(t, err)
	require.Equal(t, "https://practicum.yandex.ru/", got)
}

func TestShortener_Shorten_TrimsSpaces(t *testing.T) {
	store := repository.NewMemStore()
	gen := fixedIDGen{id: "id"}

	svc := service.NewShortener(store, gen, "http://localhost:8080/") // with trailing slash

	short, err := svc.Shorten(context.Background(), "  https://example.com/x  \n")
	require.NoError(t, err)
	require.Equal(t, "http://localhost:8080/id", short)
}

func TestShortener_Shorten_BadURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"empty", ""},
		{"no_scheme", "example.com"},
		{"no_host", "https://"},
		{"invalid_scheme", "ftp://example.com/file"},
		{"garbage", "::::"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := repository.NewMemStore()
			gen := fixedIDGen{id: "id"}
			svc := service.NewShortener(store, gen, "http://localhost:8080")

			_, err := svc.Shorten(context.Background(), tt.url)
			require.Error(t, err)
		})
	}
}

func TestShortener_Shorten_IDGenError(t *testing.T) {
	store := repository.NewMemStore()
	gen := fixedIDGen{err: errors.New("boom")}
	svc := service.NewShortener(store, gen, "http://localhost:8080")

	_, err := svc.Shorten(context.Background(), "https://example.com")
	require.Error(t, err)
}

func TestShortener_Shorten_StoreSaveError(t *testing.T) {
	saveErr := errors.New("save failed")
	store := errStore{err: saveErr}
	gen := fixedIDGen{id: "id"}

	svc := service.NewShortener(store, gen, "http://localhost:8080")
	_, err := svc.Shorten(context.Background(), "https://example.com")
	require.Error(t, err)
}

func TestShortener_Resolve_OK(t *testing.T) {
	store := repository.NewMemStore()
	require.NoError(t, store.Save(context.Background(), "id", "https://example.com"))

	svc := service.NewShortener(store, fixedIDGen{id: "x"}, "http://localhost:8080")

	orig, err := svc.Resolve(context.Background(), "id")
	require.NoError(t, err)
	require.Equal(t, "https://example.com", orig)
}

func TestShortener_Resolve_BadID(t *testing.T) {
	store := repository.NewMemStore()
	svc := service.NewShortener(store, fixedIDGen{id: "x"}, "http://localhost:8080")

	tests := []string{
		"",
		"ab/cd",
		"/id",
	}

	for _, id := range tests {
		t.Run(id, func(t *testing.T) {
			_, err := svc.Resolve(context.Background(), id)
			require.Error(t, err)
		})
	}
}

func TestShortener_Resolve_NotFound(t *testing.T) {
	store := repository.NewMemStore()
	svc := service.NewShortener(store, fixedIDGen{id: "x"}, "http://localhost:8080")

	_, err := svc.Resolve(context.Background(), "missing")
	require.ErrorIs(t, err, repository.ErrNotFound)
}

type seqGen struct {
	ids []string
	i   int
}

func (g *seqGen) NewID() (string, error) {
	if g.i >= len(g.ids) {
		return g.ids[len(g.ids)-1], nil
	}
	id := g.ids[g.i]
	g.i++
	return id, nil
}

func TestShortener_Shorten_RetriesOnCollision(t *testing.T) {
	store := repository.NewMemStore()
	require.NoError(t, store.Save(context.Background(), "dup", "https://already.com"))

	gen := &seqGen{ids: []string{"dup", "uniq"}}
	svc := service.NewShortener(store, gen, "http://localhost:8080")

	short, err := svc.Shorten(context.Background(), "https://example.com")
	require.NoError(t, err)
	require.Equal(t, "http://localhost:8080/uniq", short)

	got, err := store.Get(context.Background(), "uniq")
	require.NoError(t, err)
	require.Equal(t, "https://example.com", got)
}

func TestShortener_Shorten_ExistingOriginal_ReturnsConflict(t *testing.T) {
	store := repository.NewMemStore()
	gen := fixedIDGen{id: "newid"}

	require.NoError(t, store.Save(context.Background(), "oldid", "https://example.com"))

	svc := service.NewShortener(store, gen, "http://localhost:8080")

	_, err := svc.Shorten(context.Background(), "https://example.com")
	require.Error(t, err)

	var conflictErr *service.ConflictError
	require.ErrorAs(t, err, &conflictErr)
	require.Equal(t, "http://localhost:8080/oldid", conflictErr.ShortURL)
}
