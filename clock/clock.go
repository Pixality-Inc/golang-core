package clock

import "time"

type Clock interface {
	Now() time.Time
	Sleep(duration time.Duration)
	Since(value time.Time) time.Duration
}

var Default = New()

type Impl struct{}

func (c *Impl) Now() time.Time {
	return time.Now()
}

func (c *Impl) Sleep(duration time.Duration) {
	time.Sleep(duration)
}

func (c *Impl) Since(value time.Time) time.Duration {
	return time.Since(value)
}

func New() *Impl {
	return &Impl{}
}
