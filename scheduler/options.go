package scheduler

type Option = func(scheduler *Impl)

func WithRunAtStart(runAtStart bool) Option {
	return func(scheduler *Impl) {
		scheduler.runAtStart = runAtStart
	}
}
