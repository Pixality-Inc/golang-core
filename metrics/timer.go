package metrics

import (
	"time"

	"github.com/pixality-inc/golang-core/clock"
)

// Observer observable entity
// nolint:iface
type Observer interface {
	Observe(value float64)
}

type Timer interface {
	Observe()
}

type TimerImpl struct {
	clock     clock.Clock
	startedAt time.Time
	observer  Observer
}

func NewTimer(clock clock.Clock, observer Observer) *TimerImpl {
	return &TimerImpl{
		clock:     clock,
		startedAt: clock.Now(),
		observer:  observer,
	}
}

func (t *TimerImpl) Observe() {
	t.observer.Observe(float64(t.clock.Since(t.startedAt).Milliseconds()))
}
