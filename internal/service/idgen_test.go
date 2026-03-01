package service_test

import (
	"testing"
	"unicode"

	"github.com/stretchr/testify/require"

	"github.com/Fa1ry7a1l/url-shortener/internal/service"
)

func TestRandomID_NewID_FormatAndLength(t *testing.T) {
	gen := service.NewRandomID(8)

	id, err := gen.NewID()
	require.NoError(t, err)
	require.Len(t, id, 8)

	for _, r := range id {
		ok := unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_'
		require.True(t, ok, "unexpected char %q in id %q", r, id)
	}
}

func TestRandomID_NewID_MultipleCallsProduceDifferentValues(t *testing.T) {
	gen := service.NewRandomID(8)

	seen := make(map[string]struct{})

	for i := 0; i < 10; i++ {
		id, err := gen.NewID()
		require.NoError(t, err)
		seen[id] = struct{}{}
	}

	// Очень маловероятно, что все 10 будут одинаковыми
	require.GreaterOrEqual(t, len(seen), 2)
}
