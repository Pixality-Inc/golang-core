package ticker

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

type tickerSupport struct {
	tickCalls       atomic.Int32
	hasNextCalls    atomic.Int32
	maxHasNextCalls int32
	tick            func(ctx context.Context)
	hasNext         func(ctx context.Context) bool
}

func newTickerSupport(maxHasNextCalls int32) *tickerSupport {
	support := &tickerSupport{
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

func Test_Ticker(t *testing.T) {
	t.Parallel()

	clocks := newFakeClock()

	ctx := clock.WithClock(context.Background(), clocks)

	support := newTickerSupport(3)

	ticker := New(100*time.Millisecond, support.tick, support.hasNext)

	go func() {
		_ = ticker.Start(ctx) // nolint:errcheck
	}()

	clocks.Advance(time.Now())

	require.Equal(t, int32(3), support.tickCalls.Load())
	require.Equal(t, int32(3), support.hasNextCalls.Load())
}
