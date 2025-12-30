package clock_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/pixality-inc/golang-core/clock"
)

type fakeClock struct {
	now         time.Time
	sleepCalls  []time.Duration
	sinceResult time.Duration
}

func (f *fakeClock) Now() time.Time {
	return f.now
}

func (f *fakeClock) Sleep(d time.Duration) {
	f.sleepCalls = append(f.sleepCalls, d)
}

func (f *fakeClock) Since(_ time.Time) time.Duration {
	return f.sinceResult
}

func TestImpl_Now(t *testing.T) {
	t.Parallel()

	testClock := clock.New()

	now := testClock.Now()

	require.LessOrEqual(t, time.Since(now), time.Second,
		"Now() returned too old time")
}

func TestImpl_Sleep(t *testing.T) {
	t.Parallel()

	testClock := clock.New()

	start := time.Now()

	testClock.Sleep(10 * time.Millisecond)

	elapsed := time.Since(start)

	require.GreaterOrEqual(t, elapsed, 10*time.Millisecond,
		"Sleep() slept too little")
}

func TestImpl_Since(t *testing.T) {
	t.Parallel()

	testClock := clock.New()

	start := time.Now()

	time.Sleep(5 * time.Millisecond)

	require.GreaterOrEqual(t, testClock.Since(start), 5*time.Millisecond,
		"Since() returned too small duration")
}

func TestGetClock_Default(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	testClock := clock.GetClock(ctx)

	require.Same(t, clock.Default, testClock, "expected Default clock")
}

func TestWithClock_OverridesDefault(t *testing.T) {
	t.Parallel()

	fake := &fakeClock{
		now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	ctx := clock.WithClock(context.Background(), fake)

	testClock := clock.GetClock(ctx)

	require.Same(t, fake, testClock, "expected fake clock")
}

func TestWithClock_ClockBehavior(t *testing.T) {
	t.Parallel()

	fakeNow := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	fake := &fakeClock{
		now:         fakeNow,
		sinceResult: 42 * time.Second,
	}

	ctx := clock.WithClock(context.Background(), fake)
	testClock := clock.GetClock(ctx)

	require.True(t, testClock.Now().Equal(fakeNow), "Now(): unexpected value")

	testClock.Sleep(100 * time.Millisecond)

	require.Len(t, fake.sleepCalls, 1, "Sleep(): unexpected number of calls")

	require.Equal(t, 100*time.Millisecond, fake.sleepCalls[0], "Sleep(): unexpected duration")

	require.Equal(t, 42*time.Second, testClock.Since(time.Time{}), "Since(): unexpected duration")
}
