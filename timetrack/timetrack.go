package timetrack

import (
	"time"
)

type TimeTracker struct {
	Start time.Time
	End   time.Time
}

func New() *TimeTracker {
	return &TimeTracker{
		Start: time.Now(),
	}
}

func (t *TimeTracker) Finish() time.Duration {
	t.End = time.Now()

	return t.Duration()
}

func (t *TimeTracker) Duration() time.Duration {
	return t.End.Sub(t.Start)
}
