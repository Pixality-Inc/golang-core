package preloader

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var errTestError = errors.New("test error")

type incrementalPreloaderGetter struct {
	value int
	mutex sync.Mutex
}

func newIncrementalPreloaderGetter() *incrementalPreloaderGetter {
	return &incrementalPreloaderGetter{
		value: 0,
		mutex: sync.Mutex{},
	}
}

func (i *incrementalPreloaderGetter) nextValue() int {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	currentValue := i.value

	i.value += 1

	return currentValue
}

func TestPreloader(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	preloader := New[string](
		NewConfig("test", 10*time.Millisecond),
		"__default__",
		func(_ context.Context) (string, error) {
			return time.Now().String(), nil
		},
	)

	value1, err := preloader.Value(ctx)
	require.NoError(t, err)

	value2, err := preloader.Value(ctx)
	require.NoError(t, err)
	require.Equal(t, value2, value1)

	value3, err := preloader.Refresh(ctx)
	require.NoError(t, err)
	require.NotEqual(t, value3, value2)

	time.Sleep(11 * time.Millisecond)

	newValue, err := preloader.Value(ctx)
	require.NoError(t, err)
	require.NotEqual(t, value3, newValue)
}

func TestPreloaderFail(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	preloader := New[string](
		NewConfig("test", 10*time.Millisecond),
		"__default__",
		func(_ context.Context) (string, error) {
			return "", errTestError
		},
	)

	value1, err := preloader.Value(ctx)
	require.ErrorIs(t, err, ErrGetValue)
	require.Equal(t, "__default__", value1)

	value2, err := preloader.Refresh(ctx)
	require.ErrorIs(t, err, ErrRefreshValue)
	require.Equal(t, "__default__", value2)
}

func TestPreloaderIncremental(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	incrementalPreloader := newIncrementalPreloaderGetter()

	defaultValue := -1

	preloader := New[int](
		NewConfig("test", 10*time.Millisecond),
		defaultValue,
		func(_ context.Context) (int, error) {
			value := incrementalPreloader.nextValue()

			if value%2 == 0 {
				return value, nil
			} else {
				return defaultValue, errTestError
			}
		},
	)

	value1, err := preloader.Value(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, value1)

	value2, err := preloader.Refresh(ctx)
	require.ErrorIs(t, err, ErrRefreshValue)
	require.ErrorIs(t, err, errTestError)
	require.Equal(t, defaultValue, value2)

	value3, err := preloader.Value(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, value3)

	value4, err := preloader.Refresh(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, value4)

	value5, err := preloader.Value(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, value5)
}

func TestPreloaderConcurrentExpiredValueLoadsOnce(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	var calls atomic.Int32

	release := make(chan struct{})

	preloader := New[int](
		NewConfig("test", time.Hour),
		-1,
		func(_ context.Context) (int, error) {
			<-release

			return int(calls.Add(1)), nil
		},
	)

	const callers = 20

	values := make([]int, callers)
	errs := make([]error, callers)

	var wg sync.WaitGroup

	for i := range callers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			values[i], errs[i] = preloader.Value(ctx)
		}()
	}

	close(release)
	wg.Wait()

	require.Equal(t, int32(1), calls.Load())

	for i := range callers {
		require.NoError(t, errs[i])
		require.Equal(t, 1, values[i])
	}
}

func TestPreloaderConcurrentValueAndRefresh(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	incrementalPreloader := newIncrementalPreloaderGetter()

	preloader := New[int](
		NewConfig("test", time.Microsecond),
		-1,
		func(_ context.Context) (int, error) {
			return incrementalPreloader.nextValue(), nil
		},
	)

	var wg sync.WaitGroup

	for range 8 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for range 200 {
				_, err := preloader.Value(ctx)
				require.NoError(t, err)

				_ = preloader.LastRefreshAt()
			}
		}()
	}

	for range 2 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for range 100 {
				_, err := preloader.Refresh(ctx)
				require.NoError(t, err)
			}
		}()
	}

	wg.Wait()
}
