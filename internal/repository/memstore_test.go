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

func TestMemStore_SaveOverwrites(t *testing.T) {
	st := repository.NewMemStore()

	require.NoError(t, st.Save(context.Background(), "id1", "https://a.com"))
	require.NoError(t, st.Save(context.Background(), "id1", "https://b.com"))

	got, err := st.Get(context.Background(), "id1")
	require.NoError(t, err)
	require.Equal(t, "https://b.com", got)
}

func TestMemStore_GetNotFound(t *testing.T) {
	st := repository.NewMemStore()

	_, err := st.Get(context.Background(), "missing")
	require.ErrorIs(t, err, repository.ErrNotFound)
}
