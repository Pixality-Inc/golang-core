package future

import (
	"context"
	"errors"
	"fmt"

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
	options *Options
}

func New[T any](
	ctx context.Context,
	body func(ctx context.Context) (T, error),
	options ...Option,
) Future[T] {
	impl := &Impl[T]{
		promise: promise.New[T](),
		body:    body,
		options: NewDefaultOptions(),
	}

	for _, option := range options {
		option(impl.options)
	}

	impl.execute(ctx)

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

func (f *Impl[T]) execute(ctx context.Context) {
	err := f.options.poolExecutor.Execute(ctx, f.run)
	if err != nil {
		logger.GetLogger(ctx).WithError(err).Error("failed to execute future with pool executor")
	}
}

func (f *Impl[T]) run(ctx context.Context) error {
	value, err := f.body(ctx)
	if err != nil {
		if rErr := f.promise.Reject(err); rErr != nil {
			return fmt.Errorf("future reject: %w", errors.Join(rErr, err))
		}

		return err
	}

	if rErr := f.promise.Resolve(value); rErr != nil {
		return fmt.Errorf("future resolve: %w", rErr)
	}

	return nil
}
