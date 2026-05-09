package pool

import "context"

type taskContext struct {
	ctx  context.Context
	task Task
}
