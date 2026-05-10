package pool

import "context"

type TaskFunc = func(ctx context.Context) error

type Task interface {
	Run(ctx context.Context) error
}

type TaskImpl struct {
	function TaskFunc
}

func NewTask(function TaskFunc) Task {
	return &TaskImpl{
		function: function,
	}
}

func (t *TaskImpl) Run(ctx context.Context) error {
	return t.function(ctx)
}
