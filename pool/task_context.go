package pool

import "context"

type taskContext struct {
	ctx  context.Context //nolint:containedctx
	task Task
}
