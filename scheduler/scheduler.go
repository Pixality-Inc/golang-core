package scheduler

import (
	"context"
	"errors"
	"time"

	"github.com/pixality-inc/golang-core/clock"
)

var errContext = errors.New("context error")

type Scheduler interface {
	Start(ctx context.Context) error
}

type Impl struct {
	duration   time.Duration
	tick       func(ctx context.Context)
	hasNext    func(ctx context.Context) bool
	runAtStart bool
}

func New(
	duration time.Duration,
	tick func(ctx context.Context),
	hasNext func(ctx context.Context) bool,
	options ...Option,
) Scheduler {
	if hasNext == nil {
		hasNext = func(ctx context.Context) bool {
			return false
		}
	}

	impl := &Impl{
		duration:   duration,
		tick:       tick,
		hasNext:    hasNext,
		runAtStart: false,
	}

	for _, opt := range options {
		opt(impl)
	}

	return impl
}

func NewFromHandler(
	duration time.Duration,
	handler Handler,
	options ...Option,
) Scheduler {
	return New(duration, handler.Tick, handler.HasNext, options...)
}

func (t *Impl) Start(ctx context.Context) error {
	clocks := clock.GetClock(ctx)

	loop := func() error {
		for {
			if err := ctx.Err(); err != nil {
				return errors.Join(errContext, err)
			}

			t.tick(ctx)

			if err := ctx.Err(); err != nil {
				return errors.Join(errContext, err)
			}

			if !t.hasNext(ctx) {
				break
			}
		}

		return nil
	}

	if t.runAtStart {
		if err := loop(); err != nil {
			return err
		}
	}

	for {
		select {
		case <-ctx.Done():
			return errors.Join(errContext, ctx.Err())

		case <-clocks.After(t.duration):
			if err := loop(); err != nil {
				return err
			}
		}
	}
}
