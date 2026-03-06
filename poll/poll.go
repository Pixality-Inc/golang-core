package poll

import (
	"context"
	"errors"
	"time"

	"github.com/pixality-inc/golang-core/clock"
)

var ErrNoValue = errors.New("no value")

type NoValue = struct{}

type CheckFunc[T any] = func(ctx context.Context) (T, error)

type UntypedCheckFunc = func(ctx context.Context) error

type Poll[T any] interface {
	Poll(ctx context.Context) chan T
}

type Impl[T any] struct {
	interval time.Duration
	check    CheckFunc[T]
}

func New[T any](interval time.Duration, check CheckFunc[T]) Poll[T] {
	return &Impl[T]{
		interval: interval,
		check:    check,
	}
}

func NewUntyped(interval time.Duration, check UntypedCheckFunc) Poll[NoValue] {
	return New(interval, func(ctx context.Context) (NoValue, error) {
		if err := check(ctx); err != nil {
			return NoValue{}, err
		}

		return NoValue{}, nil
	})
}

func (p *Impl[T]) Poll(ctx context.Context) chan T {
	clocks := clock.GetClock(ctx)

	ch := make(chan T, 1)

	go func() {
		defer close(ch)

		for {
			select {
			case <-ctx.Done():
				return
			case <-clocks.After(p.interval):
				if err := ctx.Err(); err != nil {
					return
				}

				result, err := p.check(ctx)
				ctxErr := ctx.Err()

				switch {
				case ctxErr != nil:
					return
				case errors.Is(err, ErrNoValue):
					continue
				case err != nil:
					return
				default:
					ch <- result
				}
			}
		}
	}()

	return ch
}
