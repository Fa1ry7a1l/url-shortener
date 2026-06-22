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
	got.text = "state"
	p.Put(got)

	reused := p.Get()
	require.Same(t, got, reused)
	require.Empty(t, reused.text)
	require.Equal(t, int32(1), created.Load())
}
