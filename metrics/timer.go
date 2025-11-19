package metrics

import "time"

// Observer observable entity
// nolint:iface
type Observer interface {
	Observe(value float64)
}

type Timer interface {
	Observe()
}

type TimerImpl struct {
	startedAt time.Time
	observer  Observer
}

func NewTimer(observer Observer) *TimerImpl {
	return &TimerImpl{
		startedAt: time.Now(),
		observer:  observer,
	}
}

func (t *TimerImpl) Observe() {
	t.observer.Observe(float64(time.Since(t.startedAt).Milliseconds()))
}
