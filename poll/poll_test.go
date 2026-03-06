package poll

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

type pollSupport[T any] struct {
	calls       atomic.Int32
	returnAfter int32
	value       T
}

func newPollSupport[T any](value T, returnAfter int32) *pollSupport[T] {
	return &pollSupport[T]{
		calls:       atomic.Int32{},
		returnAfter: returnAfter,
		value:       value,
	}
}

func (p *pollSupport[T]) check(ctx context.Context) (T, error) {
	if p.calls.Add(1) > p.returnAfter {
		return p.value, nil
	}

	return p.value, ErrNoValue
}

func Test_Poll(t *testing.T) {
	t.Parallel()

	clocks := newFakeClock()
	ctx := clock.WithClock(context.Background(), clocks)

	support := newPollSupport(100500, 3)

	poll := New(100*time.Millisecond, support.check)

	ch := poll.Poll(ctx)

	go func() {
		for range 4 {
			clocks.Advance(time.Now())
		}
	}()

	result, ok := <-ch

	require.True(t, ok)
	require.Equal(t, 100500, result)
	require.Equal(t, int32(4), support.calls.Load())
}
