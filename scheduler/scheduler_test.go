package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/pixality-inc/golang-core/clock"
	"github.com/stretchr/testify/require"
)

type fakeClock struct {
	clock.Clock

	afterCalls chan time.Duration
	ticks      chan time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{
		afterCalls: make(chan time.Duration, 16),
		ticks:      make(chan time.Time),
	}
}

func (c *fakeClock) After(duration time.Duration) <-chan time.Time {
	c.afterCalls <- duration

	return c.ticks
}

func (c *fakeClock) waitForAfter(t *testing.T) time.Duration {
	t.Helper()

	select {
	case duration := <-c.afterCalls:
		return duration
	case <-time.After(time.Second):
		require.FailNow(t, "scheduler did not create a timer")

		return 0
	}
}

func (c *fakeClock) advance(t *testing.T) {
	t.Helper()

	select {
	case c.ticks <- time.Time{}:
	case <-time.After(time.Second):
		require.FailNow(t, "scheduler is not waiting for a timer")
	}
}

type handlerFuncs struct {
	tick    func(ctx context.Context)
	hasNext func(ctx context.Context) bool
}

func (h *handlerFuncs) Tick(ctx context.Context) {
	h.tick(ctx)
}

func (h *handlerFuncs) HasNext(ctx context.Context) bool {
	return h.hasNext(ctx)
}

type schedulerRun struct {
	cancel context.CancelFunc
	done   chan struct{}
	err    error
}

func startScheduler(t *testing.T, parent context.Context, scheduler Scheduler) *schedulerRun {
	t.Helper()

	ctx, cancel := context.WithCancel(parent)
	run := &schedulerRun{
		cancel: cancel,
		done:   make(chan struct{}),
	}

	go func() {
		defer close(run.done)

		run.err = scheduler.Start(ctx)
	}()

	t.Cleanup(func() {
		run.cancel()
		require.ErrorIs(t, run.wait(t), context.Canceled)
	})

	return run
}

func (r *schedulerRun) wait(t *testing.T) error {
	t.Helper()

	select {
	case <-r.done:
		return r.err
	case <-time.After(time.Second):
		require.FailNow(t, "scheduler did not stop")

		return nil
	}
}

type callbackCall struct {
	name  string
	value any
}

func drainCalls(calls <-chan callbackCall) []callbackCall {
	result := make([]callbackCall, 0, len(calls))

	for {
		select {
		case call := <-calls:
			result = append(result, call)
		default:
			return result
		}
	}
}

func TestSchedulerRunsBatchAfterInterval(t *testing.T) {
	t.Parallel()

	clocks := newFakeClock()
	calls := make(chan callbackCall, 16)
	hasNextResults := []bool{true, true, false, false}
	hasNextCall := 0

	type contextKey struct{}

	ctx := context.WithValue(
		clock.WithClock(context.Background(), clocks),
		contextKey{},
		"context value",
	)

	scheduler := New(
		100*time.Millisecond,
		func(ctx context.Context) {
			calls <- callbackCall{name: "tick", value: ctx.Value(contextKey{})}
		},
		func(ctx context.Context) bool {
			calls <- callbackCall{name: "has next", value: ctx.Value(contextKey{})}

			result := hasNextResults[hasNextCall]
			hasNextCall++

			return result
		},
	)
	run := startScheduler(t, ctx, scheduler)

	require.Equal(t, 100*time.Millisecond, clocks.waitForAfter(t))
	require.Empty(t, drainCalls(calls), "callbacks ran before the first interval")

	clocks.advance(t)

	require.Equal(t, 100*time.Millisecond, clocks.waitForAfter(t))
	require.Equal(t, []callbackCall{
		{name: "tick", value: "context value"},
		{name: "has next", value: "context value"},
		{name: "tick", value: "context value"},
		{name: "has next", value: "context value"},
		{name: "tick", value: "context value"},
		{name: "has next", value: "context value"},
	}, drainCalls(calls))

	clocks.advance(t)

	require.Equal(t, 100*time.Millisecond, clocks.waitForAfter(t))
	require.Equal(t, []callbackCall{
		{name: "tick", value: "context value"},
		{name: "has next", value: "context value"},
	}, drainCalls(calls), "each interval must run Tick at least once")

	run.cancel()
	require.ErrorIs(t, run.wait(t), context.Canceled)
}

func TestSchedulerRunsHandlerAtStart(t *testing.T) {
	t.Parallel()

	clocks := newFakeClock()
	calls := make(chan callbackCall, 4)
	hasNextCalls := 0
	handler := &handlerFuncs{
		tick: func(_ context.Context) {
			calls <- callbackCall{name: "tick"}
		},
		hasNext: func(_ context.Context) bool {
			calls <- callbackCall{name: "has next"}

			hasNextCalls++

			return hasNextCalls == 1
		},
	}

	scheduler := NewFromHandler(
		time.Minute,
		handler,
		WithRunAtStart(true),
	)
	run := startScheduler(t, clock.WithClock(context.Background(), clocks), scheduler)

	require.Equal(t, time.Minute, clocks.waitForAfter(t))
	require.Equal(t, []callbackCall{
		{name: "tick"},
		{name: "has next"},
		{name: "tick"},
		{name: "has next"},
	}, drainCalls(calls))

	run.cancel()
	require.ErrorIs(t, run.wait(t), context.Canceled)
}

func TestSchedulerStopsWhileWaiting(t *testing.T) {
	t.Parallel()

	clocks := newFakeClock()
	calls := make(chan callbackCall, 2)
	scheduler := New(
		time.Minute,
		func(_ context.Context) {
			calls <- callbackCall{name: "tick"}
		},
		func(_ context.Context) bool {
			calls <- callbackCall{name: "has next"}

			return false
		},
		WithRunAtStart(false),
	)
	run := startScheduler(t, clock.WithClock(context.Background(), clocks), scheduler)

	require.Equal(t, time.Minute, clocks.waitForAfter(t))
	run.cancel()

	require.ErrorIs(t, run.wait(t), context.Canceled)
	require.Empty(t, drainCalls(calls))
}

func TestSchedulerStopsBatchWhenTickCancelsContext(t *testing.T) {
	t.Parallel()

	clocks := newFakeClock()
	calls := make(chan callbackCall, 2)
	ctx, cancel := context.WithCancel(clock.WithClock(context.Background(), clocks))
	scheduler := New(
		time.Minute,
		func(_ context.Context) {
			calls <- callbackCall{name: "tick"}

			cancel()
		},
		func(_ context.Context) bool {
			calls <- callbackCall{name: "has next"}

			return true
		},
	)
	run := startScheduler(t, ctx, scheduler)

	require.Equal(t, time.Minute, clocks.waitForAfter(t))
	clocks.advance(t)

	require.ErrorIs(t, run.wait(t), context.Canceled)
	require.Equal(t, []callbackCall{{name: "tick"}}, drainCalls(calls))
}

func TestSchedulerStopsBatchWhenHasNextCancelsContext(t *testing.T) {
	t.Parallel()

	clocks := newFakeClock()
	calls := make(chan callbackCall, 3)
	ctx, cancel := context.WithCancel(clock.WithClock(context.Background(), clocks))
	scheduler := New(
		time.Minute,
		func(_ context.Context) {
			calls <- callbackCall{name: "tick"}
		},
		func(_ context.Context) bool {
			calls <- callbackCall{name: "has next"}

			cancel()

			return true
		},
	)
	run := startScheduler(t, ctx, scheduler)

	require.Equal(t, time.Minute, clocks.waitForAfter(t))
	clocks.advance(t)

	require.ErrorIs(t, run.wait(t), context.Canceled)
	require.Equal(t, []callbackCall{
		{name: "tick"},
		{name: "has next"},
	}, drainCalls(calls))
}

func TestSchedulerDoesNotRunWithCanceledContext(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name       string
		runAtStart bool
	}{
		{name: "wait for interval", runAtStart: false},
		{name: "run at start", runAtStart: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			clocks := newFakeClock()
			calls := make(chan callbackCall, 2)
			ctx, cancel := context.WithCancel(clock.WithClock(context.Background(), clocks))
			cancel()

			scheduler := New(
				time.Minute,
				func(_ context.Context) {
					calls <- callbackCall{name: "tick"}
				},
				func(_ context.Context) bool {
					calls <- callbackCall{name: "has next"}

					return false
				},
				WithRunAtStart(testCase.runAtStart),
			)
			run := startScheduler(t, ctx, scheduler)

			require.ErrorIs(t, run.wait(t), context.Canceled)
			require.Empty(t, drainCalls(calls))
		})
	}
}
