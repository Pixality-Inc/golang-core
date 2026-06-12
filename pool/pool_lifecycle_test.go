package pool

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pixality-inc/golang-core/errors"
)

var errTask = errors.New("test.task", "task failed")

func TestPoolExecuteBeforeStart(t *testing.T) {
	t.Parallel()

	poolExecutor := New("test", 1)

	err := poolExecutor.Execute(t.Context(), func(_ context.Context) error { return nil })
	require.ErrorIs(t, err, ErrNotStarted)

	err = poolExecutor.ExecuteTask(t.Context(), NewTask(func(_ context.Context) error { return nil }))
	require.ErrorIs(t, err, ErrNotStarted)
}

func TestPoolStartTwice(t *testing.T) {
	t.Parallel()

	poolExecutor := New("test", 1)

	require.NoError(t, poolExecutor.Start(t.Context()))
	require.ErrorIs(t, poolExecutor.Start(t.Context()), ErrAlreadyStarted)
	require.NoError(t, poolExecutor.Stop())
}

func TestPoolStopWithoutStart(t *testing.T) {
	t.Parallel()

	poolExecutor := New("test", 1)

	require.ErrorIs(t, poolExecutor.Stop(), ErrAlreadyStopped)
}

func TestPoolStopTwice(t *testing.T) {
	t.Parallel()

	poolExecutor := New("test", 1)

	require.NoError(t, poolExecutor.Start(t.Context()))
	require.NoError(t, poolExecutor.Stop())
	require.ErrorIs(t, poolExecutor.Stop(), ErrAlreadyStopped)
}

func TestPoolExecuteAfterStop(t *testing.T) {
	t.Parallel()

	poolExecutor := New("test", 1)

	require.NoError(t, poolExecutor.Start(t.Context()))
	require.NoError(t, poolExecutor.Stop())

	err := poolExecutor.Execute(t.Context(), func(_ context.Context) error { return nil })
	require.ErrorIs(t, err, ErrNotStarted)
}

func TestPoolRestart(t *testing.T) {
	t.Parallel()

	poolExecutor := New("test", 1)

	require.NoError(t, poolExecutor.Start(t.Context()))
	require.NoError(t, poolExecutor.Stop())
	require.NoError(t, poolExecutor.Start(t.Context()))

	values := make(chan int, 1)
	executeErr := make(chan error, 1)

	go func() {
		executeErr <- poolExecutor.Execute(t.Context(), func(_ context.Context) error {
			values <- 1

			return nil
		})
	}()

	require.NoError(t, receiveError(t, executeErr))
	require.Equal(t, 1, receiveValue(t, values))
	require.NoError(t, poolExecutor.Stop())
}

func TestPoolRunsTasksConcurrently(t *testing.T) {
	t.Parallel()

	poolExecutor := New("test", 3)

	require.NoError(t, poolExecutor.Start(t.Context()))

	started := make(chan struct{}, 3)
	release := make(chan struct{})
	finished := make(chan struct{}, 3)

	task := func(_ context.Context) error {
		started <- struct{}{}

		<-release

		finished <- struct{}{}

		return nil
	}

	executeErr := make(chan error, 1)

	go func() {
		executeErr <- poolExecutor.Execute(t.Context(), task, task, task)
	}()

	for range 3 {
		receiveValue(t, started)
	}

	close(release)

	for range 3 {
		receiveValue(t, finished)
	}

	require.NoError(t, receiveError(t, executeErr))
	require.NoError(t, poolExecutor.Stop())
}

func TestPoolWorkerContinuesAfterTaskError(t *testing.T) {
	t.Parallel()

	poolExecutor := New("test", 1)

	require.NoError(t, poolExecutor.Start(t.Context()))

	values := make(chan int, 1)
	executeErr := make(chan error, 1)

	go func() {
		executeErr <- poolExecutor.Execute(
			t.Context(),
			func(_ context.Context) error {
				return errTask
			},
			func(_ context.Context) error {
				values <- 1

				return nil
			},
		)
	}()

	require.NoError(t, receiveError(t, executeErr))
	require.Equal(t, 1, receiveValue(t, values))
	require.NoError(t, poolExecutor.Stop())
}

func TestPoolExecuteTask(t *testing.T) {
	t.Parallel()

	poolExecutor := New("test", 1)

	require.NoError(t, poolExecutor.Start(t.Context()))

	values := make(chan int, 2)
	executeErr := make(chan error, 1)

	go func() {
		executeErr <- poolExecutor.ExecuteTask(
			t.Context(),
			NewTask(func(_ context.Context) error {
				values <- 1

				return nil
			}),
			NewTask(func(_ context.Context) error {
				values <- 2

				return nil
			}),
		)
	}()

	require.NoError(t, receiveError(t, executeErr))
	require.Equal(t, 1, receiveValue(t, values))
	require.Equal(t, 2, receiveValue(t, values))
	require.NoError(t, poolExecutor.Stop())
}

func TestPoolWorkersExitOnPoolContextCancellation(t *testing.T) {
	t.Parallel()

	poolExecutor := New("test", 2)

	poolCtx, cancelPool := context.WithCancel(t.Context())
	defer cancelPool()

	require.NoError(t, poolExecutor.Start(poolCtx))

	started := make(chan struct{})
	release := make(chan struct{})
	completed := make(chan int, 1)

	require.NoError(t, poolExecutor.Execute(t.Context(), func(_ context.Context) error {
		close(started)

		<-release

		completed <- 1

		return nil
	}))

	waitForClosed(t, started)
	cancelPool()
	close(release)

	require.Equal(t, 1, receiveValue(t, completed))
	require.NoError(t, poolExecutor.Stop())
}

func TestDefaultPoolLifecycle(t *testing.T) {
	t.Parallel()

	defaultPool := NewDefault()

	require.NoError(t, defaultPool.Start(t.Context()))
	require.NoError(t, defaultPool.Stop())
}

func TestDefaultPoolExecute(t *testing.T) {
	t.Parallel()

	values := make(chan int, 1)

	require.NoError(t, Default.Execute(t.Context(), func(_ context.Context) error {
		values <- 1

		return nil
	}))

	require.Equal(t, 1, receiveValue(t, values))
}

func TestDefaultPoolExecuteTaskWithError(t *testing.T) {
	t.Parallel()

	executed := make(chan struct{})

	require.NoError(t, Default.ExecuteTask(t.Context(), NewTask(func(_ context.Context) error {
		close(executed)

		return errTask
	})))

	waitForClosed(t, executed)
}

func TestNewTaskRun(t *testing.T) {
	t.Parallel()

	task := NewTask(func(_ context.Context) error { return errTask })
	require.ErrorIs(t, task.Run(t.Context()), errTask)

	okTask := NewTask(func(_ context.Context) error { return nil })
	require.NoError(t, okTask.Run(t.Context()))
}
