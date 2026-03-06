package scheduler

import "context"

type Handler interface {
	Tick(ctx context.Context)
	HasNext(ctx context.Context) bool
}
