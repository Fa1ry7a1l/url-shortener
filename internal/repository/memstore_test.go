package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Fa1ry7a1l/url-shortener/internal/repository"
)

func TestMemStore_SaveAndGet(t *testing.T) {
	st := repository.NewMemStore()

	err := st.Save(context.Background(), "id1", "https://example.com")
	require.NoError(t, err)

	got, err := st.Get(context.Background(), "id1")
	require.NoError(t, err)
	require.Equal(t, "https://example.com", got)
}

func TestMemStore_SaveSameID_ReturnsErrIDExists(t *testing.T) {
	st := repository.NewMemStore()

	require.NoError(t, st.Save(context.Background(), "id1", "https://a.com"))
	err := st.Save(context.Background(), "id1", "https://b.com")
	require.ErrorIs(t, err, repository.ErrIDExists)

	got, err := st.Get(context.Background(), "id1")
	require.NoError(t, err)
	require.Equal(t, "https://a.com", got)
}

func TestMemStore_GetNotFound(t *testing.T) {
	st := repository.NewMemStore()

	_, err := st.Get(context.Background(), "missing")
	require.ErrorIs(t, err, repository.ErrNotFound)
}

func TestMemStore_DeleteBatchByUser(t *testing.T) {
	st := repository.NewMemStore()
	require.NoError(t, st.SaveForUser(context.Background(), "id1", "https://one.example", "user-1"))
	require.NoError(t, st.SaveForUser(context.Background(), "id2", "https://two.example", "user-2"))

	require.NoError(t, st.DeleteBatchByUser(context.Background(), "user-1", []string{"id1", "id2"}))

	_, err := st.Get(context.Background(), "id1")
	require.ErrorIs(t, err, repository.ErrDeleted)

	got, err := st.Get(context.Background(), "id2")
	require.NoError(t, err)
	require.Equal(t, "https://two.example", got)
}

func TestMemStore_GetByUserSkipsDeletedAndOtherUsers(t *testing.T) {
	st := repository.NewMemStore()
	require.NoError(t, st.SaveForUser(context.Background(), "id1", "https://one.example", "user-1"))
	require.NoError(t, st.SaveForUser(context.Background(), "id2", "https://two.example", "user-1"))
	require.NoError(t, st.SaveForUser(context.Background(), "id3", "https://three.example", "user-2"))
	require.NoError(t, st.DeleteBatchByUser(context.Background(), "user-1", []string{"id2"}))

	got, err := st.GetByUser(context.Background(), "user-1")
	require.NoError(t, err)
	require.ElementsMatch(t, []repository.UserURL{
		{ID: "id1", Original: "https://one.example"},
	}, got)
}

func TestMemStore_Stats(t *testing.T) {
	st := repository.NewMemStore()

	require.NoError(t, st.Save(context.Background(), "public", "https://public.example"))
	require.NoError(t, st.SaveForUser(context.Background(), "id1", "https://one.example", "user-1"))
	require.NoError(t, st.SaveForUser(context.Background(), "id2", "https://two.example", "user-1"))
	require.NoError(t, st.SaveForUser(context.Background(), "id3", "https://three.example", "user-2"))
	require.NoError(t, st.DeleteBatchByUser(context.Background(), "user-1", []string{"id2"}))

	stats, err := st.Stats(context.Background())
	require.NoError(t, err)
	require.Equal(t, repository.Stats{URLs: 4, Users: 2}, stats)
}
