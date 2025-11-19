package cached_value

import (
	"context"
	"errors"
	"sync"
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
