// Package pool provides a type-safe wrapper around sync.Pool.
package pool

import "sync"

// Resetter is implemented by values that can reset their state.
type Resetter interface {
	Reset()
}

// Pool stores reusable values of type T.
type Pool[T Resetter] struct {
	pool sync.Pool
}

// New creates a Pool that uses newFn to create values when the pool is empty.
func New[T Resetter](newFn func() T) *Pool[T] {
	return &Pool[T]{
		pool: sync.Pool{
			New: func() any {
				return newFn()
			},
		},
	}
}

// Get returns a value from the pool.
func (p *Pool[T]) Get() T {
	return p.pool.Get().(T)
}

// Put resets value and returns it to the pool.
func (p *Pool[T]) Put(value T) {
	value.Reset()
	p.pool.Put(value)
}
