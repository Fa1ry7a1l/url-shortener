package pool

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

type value struct {
	text string
}

func (v *value) Reset() {
	v.text = ""
}

func TestPool(t *testing.T) {
	var created atomic.Int32
	p := New(func() *value {
		created.Add(1)
		return &value{}
	})

	got := p.Get()
	require.NotNil(t, got)
	require.Equal(t, int32(1), created.Load())

	got.text = "state"
	p.Put(got)
	require.Empty(t, got.text)

	reused := p.Get()
	require.NotNil(t, reused)
	require.Empty(t, reused.text)
}
