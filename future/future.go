package future

import (
	"context"

	"github.com/pixality-inc/golang-core/either"
	"github.com/pixality-inc/golang-core/logger"
	"github.com/pixality-inc/golang-core/promise"
)

type Future[T any] interface {
	Get(ctx context.Context) (T, error)
	Chan() <-chan either.EitherError[T]
	IsResolved() bool
}

type Impl[T any] struct {
	promise promise.Promise[T]
	body    func(ctx context.Context) (T, error)
}

func New[T any](ctx context.Context, body func(ctx context.Context) (T, error)) Future[T] {
	impl := &Impl[T]{
		promise: promise.New[T](),
		body:    body,
	}

	go impl.run(ctx)

	return impl
}

func (f *Impl[T]) Get(ctx context.Context) (T, error) {
	return f.promise.Get(ctx)
}

func (f *Impl[T]) Chan() <-chan either.EitherError[T] {
	return f.promise.Chan()
}

func (f *Impl[T]) IsResolved() bool {
	return f.promise.IsResolved()
}

func (f *Impl[T]) run(ctx context.Context) {
	value, err := f.body(ctx)
	if err != nil {
		if rErr := f.promise.Reject(err); rErr != nil {
			logger.GetLogger(ctx).WithError(rErr).Error("future reject")
		}

		return
	}

	if rErr := f.promise.Resolve(value); rErr != nil {
		logger.GetLogger(ctx).WithError(rErr).Error("future resolve")
	}
}
