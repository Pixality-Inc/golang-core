package timetrack

import (
	"context"
	"time"

	"github.com/pixality-inc/golang-core/clock"
)

type TimeTracker struct {
	clock clock.Clock
	Start time.Time
	End   time.Time
}

func New(ctx context.Context) *TimeTracker {
	contextClock := clock.GetClock(ctx)

	return &TimeTracker{
		clock: contextClock,
		Start: contextClock.Now(),
	}
}

func (t *TimeTracker) Finish() time.Duration {
	t.End = t.clock.Now()

	return t.Duration()
}

func (t *TimeTracker) Duration() time.Duration {
	return t.End.Sub(t.Start)
}
