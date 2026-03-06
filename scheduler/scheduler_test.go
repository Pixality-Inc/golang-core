package scheduler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pixality-inc/golang-core/clock"
	"github.com/stretchr/testify/require"
)

type fakeClock struct {
	clock.Clock

	ch chan time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{
		ch: make(chan time.Time),
	}
}

func (c *fakeClock) After(_ time.Duration) <-chan time.Time {
	return c.ch
}

func (c *fakeClock) Advance(value time.Time) {
	c.ch <- value
}

type schedulerSupport struct {
	tickCalls       atomic.Int32
	hasNextCalls    atomic.Int32
	maxHasNextCalls int32
	tick            func(ctx context.Context)
	hasNext         func(ctx context.Context) bool
}

func newSchedulerSupport(maxHasNextCalls int32) *schedulerSupport {
	support := &schedulerSupport{
		tickCalls:       atomic.Int32{},
		hasNextCalls:    atomic.Int32{},
		maxHasNextCalls: maxHasNextCalls,
		tick:            nil,
		hasNext:         nil,
	}

	support.tick = func(ctx context.Context) {
		support.tickCalls.Add(1)
	}

	support.hasNext = func(ctx context.Context) bool {
		return support.hasNextCalls.Add(1) < support.maxHasNextCalls
	}

	return support
}

func (s *schedulerSupport) Tick(ctx context.Context) {
	s.tick(ctx)
}

func (s *schedulerSupport) HasNext(ctx context.Context) bool {
	return s.hasNext(ctx)
}

func Test_Scheduler(t *testing.T) {
	t.Parallel()

	clocks := newFakeClock()

	ctx := clock.WithClock(context.Background(), clocks)

	support := newSchedulerSupport(3)

	scheduler := NewFromHandler(100*time.Millisecond, support)

	go func() {
		_ = scheduler.Start(ctx) // nolint:errcheck
	}()

	clocks.Advance(time.Now())

	require.Equal(t, int32(2), support.tickCalls.Load())
	require.Equal(t, int32(3), support.hasNextCalls.Load())
}
