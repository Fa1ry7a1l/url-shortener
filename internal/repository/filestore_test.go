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

func TestFileStore_GetByUserSkipsDeletedAndOtherUsers(t *testing.T) {
	path := t.TempDir() + "/db.json"

	store, err := repository.NewFileStore(path)
	require.NoError(t, err)

	require.NoError(t, store.SaveForUser(context.Background(), "id1", "https://one.example", "user-1"))
	require.NoError(t, store.SaveForUser(context.Background(), "id2", "https://two.example", "user-1"))
	require.NoError(t, store.SaveForUser(context.Background(), "id3", "https://three.example", "user-2"))
	require.NoError(t, store.DeleteBatchByUser(context.Background(), "user-1", []string{"id2"}))

	got, err := store.GetByUser(context.Background(), "user-1")
	require.NoError(t, err)
	require.ElementsMatch(t, []repository.UserURL{
		{ID: "id1", Original: "https://one.example"},
	}, got)

	restarted, err := repository.NewFileStore(path)
	require.NoError(t, err)
	got, err = restarted.GetByUser(context.Background(), "user-1")
	require.NoError(t, err)
	require.ElementsMatch(t, []repository.UserURL{
		{ID: "id1", Original: "https://one.example"},
	}, got)
}

func TestFileStore_Stats(t *testing.T) {
	path := t.TempDir() + "/db.json"

	store, err := repository.NewFileStore(path)
	require.NoError(t, err)

	require.NoError(t, store.Save(context.Background(), "public", "https://public.example"))
	require.NoError(t, store.SaveForUser(context.Background(), "id1", "https://one.example", "user-1"))
	require.NoError(t, store.SaveForUser(context.Background(), "id2", "https://two.example", "user-1"))
	require.NoError(t, store.SaveForUser(context.Background(), "id3", "https://three.example", "user-2"))
	require.NoError(t, store.DeleteBatchByUser(context.Background(), "user-1", []string{"id2"}))

	stats, err := store.Stats(context.Background())
	require.NoError(t, err)
	require.Equal(t, repository.Stats{URLs: 4, Users: 2}, stats)

	restarted, err := repository.NewFileStore(path)
	require.NoError(t, err)
	stats, err = restarted.Stats(context.Background())
	require.NoError(t, err)
	require.Equal(t, repository.Stats{URLs: 4, Users: 2}, stats)
}
