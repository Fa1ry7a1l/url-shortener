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

func TestFileStore_DeleteBatchByUser(t *testing.T) {
	path := t.TempDir() + "/db.json"

	store, err := repository.NewFileStore(path)
	require.NoError(t, err)

	require.NoError(t, store.SaveForUser(context.Background(), "id1", "https://one.example", "user-1"))
	require.NoError(t, store.SaveForUser(context.Background(), "id2", "https://two.example", "user-2"))

	require.NoError(t, store.DeleteBatchByUser(context.Background(), "user-1", []string{"id1", "id2"}))

	_, err = store.Get(context.Background(), "id1")
	require.ErrorIs(t, err, repository.ErrDeleted)

	got, err := store.Get(context.Background(), "id2")
	require.NoError(t, err)
	require.Equal(t, "https://two.example", got)

	restarted, err := repository.NewFileStore(path)
	require.NoError(t, err)
	_, err = restarted.Get(context.Background(), "id1")
	require.ErrorIs(t, err, repository.ErrDeleted)
}
