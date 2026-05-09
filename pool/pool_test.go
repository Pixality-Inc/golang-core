package pool

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPoolProcessesMultipleTasksWithSingleWorker(t *testing.T) {
	t.Parallel()

	p := New("test", 1)

	require.NoError(t, p.Start(t.Context()))

	values := make(chan int, 2)
	executeErr := make(chan error, 1)

	go func() {
		executeErr <- p.Execute(
			t.Context(),
			func(_ context.Context) error {
				values <- 1

				return nil
			},
			func(_ context.Context) error {
				values <- 2

				return nil
			},
		)
	}()

	require.NoError(t, receiveError(t, executeErr))
	require.Equal(t, 1, receiveValue(t, values))
	require.Equal(t, 2, receiveValue(t, values))
	require.NoError(t, p.Stop())
}

func TestPoolWorkerContinuesAfterTaskContextCancellation(t *testing.T) {
	t.Parallel()

	p := New("test", 1)

	require.NoError(t, p.Start(t.Context()))

	taskCtx, cancelTask := context.WithCancel(t.Context())
	started := make(chan struct{})

	require.NoError(t, p.Execute(taskCtx, func(ctx context.Context) error {
		close(started)

		<-ctx.Done()

		return ctx.Err()
	}))

	waitForClosed(t, started)
	cancelTask()

	values := make(chan int, 1)
	executeErr := make(chan error, 1)

	go func() {
		executeErr <- p.Execute(t.Context(), func(_ context.Context) error {
			values <- 1

			return nil
		})
	}()

	require.NoError(t, receiveError(t, executeErr))
	require.Equal(t, 1, receiveValue(t, values))
	require.NoError(t, p.Stop())
}

func receiveError(t *testing.T, ch <-chan error) error {
	t.Helper()

	select {
	case err := <-ch:
		return err
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for pool execute")

		return nil
	}
}

func receiveValue[T any](t *testing.T, ch <-chan T) T {
	t.Helper()

	select {
	case value := <-ch:
		return value
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for pool task")

		var zero T

		return zero
	}
}

func waitForClosed(t *testing.T, ch <-chan struct{}) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for pool task start")
	}
}
