package lazy

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var (
	errLazyTest                = errors.New("lazy test error")
	errLazyContextValueMissing = errors.New("lazy context value missing")
)

func TestLazyDoesNotCallBeforeGet(t *testing.T) {
	t.Parallel()

	var called atomic.Bool

	laz := New(func(ctx context.Context) (string, error) {
		called.Store(true)

		return "ready", nil
	})

	require.False(t, called.Load())

	value, err := laz.Get(context.Background())

	require.NoError(t, err)
	require.Equal(t, "ready", value)
	require.True(t, called.Load())
}

func TestLazyCachesValue(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64

	laz := New(func(ctx context.Context) (int, error) {
		return int(calls.Add(1)), nil
	})

	value, err := laz.Get(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, value)

	value, err = laz.Get(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, value)
	require.Equal(t, int64(1), calls.Load())
}

func TestLazyCachesError(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64

	laz := New(func(ctx context.Context) (int, error) {
		calls.Add(1)

		return 0, errLazyTest
	})

	value, err := laz.Get(context.Background())
	require.ErrorIs(t, err, errLazyTest)
	require.Zero(t, value)

	value, err = laz.Get(context.Background())
	require.ErrorIs(t, err, errLazyTest)
	require.Zero(t, value)
	require.Equal(t, int64(1), calls.Load())
}

func TestLazyPassesGetContextToBody(t *testing.T) {
	t.Parallel()

	type contextKey string

	ctx := context.WithValue(context.Background(), contextKey("name"), "lazy-context")

	laz := New(func(ctx context.Context) (string, error) {
		value, ok := ctx.Value(contextKey("name")).(string)
		if !ok {
			return "", errLazyContextValueMissing
		}

		return value, nil
	})

	value, err := laz.Get(ctx)

	require.NoError(t, err)
	require.Equal(t, "lazy-context", value)
}

func TestLazyConcurrentGetCallsBodyOnce(t *testing.T) {
	t.Parallel()

	const goroutines = 10

	var calls atomic.Int64

	started := make(chan struct{})

	release := make(chan struct{})

	laz := New(func(ctx context.Context) (int, error) {
		calls.Add(1)
		close(started)
		<-release

		return 42, nil
	})

	var wg sync.WaitGroup

	results := make(chan int, goroutines)

	errs := make(chan error, goroutines)

	for range goroutines {
		wg.Go(func() {
			value, err := laz.Get(context.Background())

			results <- value

			errs <- err
		})
	}

	waitForClosed(t, started)

	require.Never(t, func() bool {
		return calls.Load() > 1
	}, 20*time.Millisecond, time.Millisecond)

	close(release)
	wg.Wait()
	close(results)
	close(errs)

	for value := range results {
		require.Equal(t, 42, value)
	}

	for err := range errs {
		require.NoError(t, err)
	}

	require.Equal(t, int64(1), calls.Load())
}

func waitForClosed(t *testing.T, ch <-chan struct{}) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for lazy body start")
	}
}
