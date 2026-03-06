package ticker

import (
	"context"
	"errors"
	"time"

	"github.com/pixality-inc/golang-core/clock"
)

var errContext = errors.New("context error")

type Ticker interface {
	Start(ctx context.Context) error
}

type Impl struct {
	duration time.Duration
	tick     func(ctx context.Context)
	hasNext  func(ctx context.Context) bool
}

func New(
	duration time.Duration,
	tick func(ctx context.Context),
	hasNext func(ctx context.Context) bool,
) Ticker {
	return &Impl{
		duration: duration,
		tick:     tick,
		hasNext:  hasNext,
	}
}

func (t *Impl) Start(ctx context.Context) error {
	clocks := clock.GetClock(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-clocks.After(t.duration):
			if err := ctx.Err(); err != nil {
				return errors.Join(errContext, err)
			}

			t.tick(ctx)

			for {
				if err := ctx.Err(); err != nil {
					return errors.Join(errContext, err)
				}

				if !t.hasNext(ctx) {
					break
				}

				t.tick(ctx)
			}
		}
	}
}
