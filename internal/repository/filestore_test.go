package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Fa1ry7a1l/url-shortener/internal/repository"
)

func TestFileStore_SaveAndLoad(t *testing.T) {
	path := t.TempDir() + "/db.json"

	store, err := repository.NewFileStore(path)
	require.NoError(t, err)

	err = store.Save(context.Background(), "abc123", "https://example.com")
	require.NoError(t, err)

	got, err := store.Get(context.Background(), "abc123")
	require.NoError(t, err)
	require.Equal(t, "https://example.com", got)

	// эмулируем рестарт сервера
	store2, err := repository.NewFileStore(path)
	require.NoError(t, err)

	got, err = store2.Get(context.Background(), "abc123")
	require.NoError(t, err)
	require.Equal(t, "https://example.com", got)
}

func TestFileStore_SaveDuplicateID(t *testing.T) {
	path := t.TempDir() + "/db.json"

	store, err := repository.NewFileStore(path)
	require.NoError(t, err)

	require.NoError(t, store.Save(context.Background(), "same", "https://a.com"))

	err = store.Save(context.Background(), "same", "https://b.com")
	require.ErrorIs(t, err, repository.ErrIDExists)
}

func TestFileStore_GetNotFound(t *testing.T) {
	path := t.TempDir() + "/db.json"

	store, err := repository.NewFileStore(path)
	require.NoError(t, err)

	_, err = store.Get(context.Background(), "missing")
	require.ErrorIs(t, err, repository.ErrNotFound)
}
