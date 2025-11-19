package temporal

import (
	"go.temporal.io/sdk/workflow"
)

type Awaitable interface {
	Name() string
	Future() workflow.Future
}

type ActivityFuture[OUT any] struct {
	name         string
	future       workflow.Future
	defaultValue OUT
}

func NewActivityFuture[OUT any](name string, future workflow.Future, defaultValue OUT) *ActivityFuture[OUT] {
	return &ActivityFuture[OUT]{
		name:         name,
		future:       future,
		defaultValue: defaultValue,
	}
}

func (f *ActivityFuture[OUT]) Get(ctx workflow.Context, outRef *OUT) error {
	return f.future.Get(ctx, outRef)
}

func (f *ActivityFuture[OUT]) Name() string {
	return f.name
}

func (f *ActivityFuture[OUT]) Future() workflow.Future {
	return f.future
}
