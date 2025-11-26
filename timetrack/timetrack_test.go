package timetrack

import (
	"context"
	"testing"
	"time"

	"github.com/pixality-inc/golang-core/clock"
	"github.com/stretchr/testify/require"
)

const sleepMillis int64 = 10

func TestTimeTrack(t *testing.T) {
	t.Parallel()

	// @todo test with clock
	testClock := clock.Default
	ctx := clock.WithClock(context.Background(), testClock)

	now := testClock.Now()

	track := New(ctx)

	startDelta := track.Start.Sub(now)

	time.Sleep(time.Duration(sleepMillis) * time.Millisecond)

	finishedDuration := track.Finish()
	duration := track.Duration()

	endDelta := time.Since(track.End)

	require.Greater(t, track.End, track.Start)
	require.GreaterOrEqual(t, endDelta.Milliseconds(), int64(0))
	require.GreaterOrEqual(t, finishedDuration.Milliseconds(), sleepMillis)
	require.GreaterOrEqual(t, duration.Milliseconds(), sleepMillis)
	require.LessOrEqual(t, endDelta.Milliseconds(), sleepMillis+1)
	require.LessOrEqual(t, startDelta.Milliseconds(), int64(1))
	require.InDelta(t, sleepMillis, finishedDuration.Milliseconds(), 1)
	require.InDelta(t, finishedDuration, duration, 0)
}
