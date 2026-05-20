package lazy

import (
	"context"
	"sync"
)

type Func[T any] func(ctx context.Context) (T, error)

type Lazy[T any] interface {
	Get(ctx context.Context) (T, error)
}

type Impl[T any] struct {
	function Func[T]
	value    T
	err      error
	once     sync.Once
}

func New[T any](function Func[T]) Lazy[T] {
	return &Impl[T]{
		function: function,
		once:     sync.Once{},
	}
}

func (l *Impl[T]) Get(ctx context.Context) (T, error) {
	l.once.Do(func() {
		l.value, l.err = l.function(ctx)
	})

	return l.value, l.err
}
