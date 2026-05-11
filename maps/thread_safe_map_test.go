package maps

import (
	"sync"
	"testing"

	"github.com/pixality-inc/golang-core/iterator"
	"github.com/stretchr/testify/require"
)

func TestNewThreadSafeMapCopiesInitialValue(t *testing.T) {
	t.Parallel()

	initialValue := map[string]int{"a": 1}

	mapImpl := NewThreadSafeMap(initialValue)
	initialValue["a"] = 2
	initialValue["b"] = 3

	value, ok := mapImpl.Get("a")
	require.True(t, ok)
	require.Equal(t, 1, value)
	require.False(t, mapImpl.HasKey("b"))
	require.Equal(t, 1, mapImpl.Len())
}

func TestNewThreadSafeMapWithNilInitialValue(t *testing.T) {
	t.Parallel()

	mapImpl := NewThreadSafeMap[string, int](nil)

	require.Equal(t, 0, mapImpl.Len())
	require.Empty(t, mapImpl.Entries())
	require.Empty(t, mapImpl.AsGoMap())

	require.NoError(t, mapImpl.Set("a", 1))
	require.Equal(t, map[string]int{"a": 1}, mapImpl.AsGoMap())
}

func TestThreadSafeMapSetGetAndDelete(t *testing.T) {
	t.Parallel()

	mapImpl := NewThreadSafeMap[string, int](nil)

	value, ok := mapImpl.Get("missing")
	require.False(t, ok)
	require.Zero(t, value)
	require.False(t, mapImpl.HasKey("missing"))
	require.Nil(t, mapImpl.GetEntry("missing"))

	require.NoError(t, mapImpl.Set("a", 1))
	require.NoError(t, mapImpl.Set("b", 2))
	require.NoError(t, mapImpl.Set("a", 10))

	value, ok = mapImpl.Get("a")
	require.True(t, ok)
	require.Equal(t, 10, value)
	require.True(t, mapImpl.HasKey("a"))
	require.Equal(t, 2, mapImpl.Len())

	entry := mapImpl.GetEntry("a")
	require.NotNil(t, entry)
	require.Equal(t, "a", entry.Key())
	require.Equal(t, 10, entry.Value())

	require.NoError(t, mapImpl.Delete("b"))
	require.False(t, mapImpl.HasKey("b"))
	require.Equal(t, 1, mapImpl.Len())

	require.NoError(t, mapImpl.Delete("missing"))
	require.Equal(t, map[string]int{"a": 10}, mapImpl.AsGoMap())
}

func TestThreadSafeMapEntries(t *testing.T) {
	t.Parallel()

	mapImpl := NewThreadSafeMap(map[string]int{"a": 1, "b": 2})

	entries := mapImpl.Entries()

	require.Len(t, entries, 2)
	require.Equal(t, map[string]int{"a": 1, "b": 2}, entriesToMap(t, entries))
}

func TestThreadSafeMapAsGoMapReturnsCopy(t *testing.T) {
	t.Parallel()

	mapImpl := NewThreadSafeMap(map[string]int{"a": 1})

	goMap := mapImpl.AsGoMap()
	goMap["a"] = 2
	goMap["b"] = 3

	value, ok := mapImpl.Get("a")
	require.True(t, ok)
	require.Equal(t, 1, value)
	require.False(t, mapImpl.HasKey("b"))
	require.Equal(t, map[string]int{"a": 1}, mapImpl.AsGoMap())
}

func TestThreadSafeMapIterator(t *testing.T) {
	t.Parallel()

	mapImpl := NewThreadSafeMap(map[string]int{"a": 1, "b": 2})

	entries, err := iterator.Materialize(mapImpl.AsIterator())

	require.NoError(t, err)
	require.Equal(t, map[string]int{"a": 1, "b": 2}, entriesToMap(t, entries))
}

func TestThreadSafeMapIteratorUsesEntriesSnapshot(t *testing.T) {
	t.Parallel()

	mapImpl := NewThreadSafeMap(map[string]int{"a": 1})
	iter := mapImpl.AsIterator()

	require.NoError(t, mapImpl.Set("b", 2))
	require.NoError(t, mapImpl.Delete("a"))

	entries, err := iterator.Materialize(iter)

	require.NoError(t, err)
	require.Equal(t, map[string]int{"a": 1}, entriesToMap(t, entries))
}

func TestThreadSafeMapConcurrentAccess(t *testing.T) {
	t.Parallel()

	const (
		workers       = 16
		keysPerWorker = 64
	)

	mapImpl := NewThreadSafeMap[int, int](nil)

	var wg sync.WaitGroup

	failures := make(chan concurrentAccessFailure, workers*keysPerWorker)

	for workerID := range workers {
		wg.Go(func() {
			for idx := range keysPerWorker {
				key := workerID*keysPerWorker + idx
				expectedValue := key * 10

				if err := mapImpl.Set(key, expectedValue); err != nil {
					failures <- concurrentAccessFailure{err: err}

					return
				}

				value, ok := mapImpl.Get(key)
				if !ok || value != expectedValue {
					failures <- concurrentAccessFailure{
						key:           key,
						value:         value,
						ok:            ok,
						expectedValue: expectedValue,
					}

					return
				}

				_ = mapImpl.Entries()
				_ = mapImpl.AsGoMap()
			}
		})
	}

	wg.Wait()
	close(failures)

	for failure := range failures {
		require.NoError(t, failure.err)
		require.Equal(t, failure.expectedValue, failure.value)
		require.True(t, failure.ok, "key %d must be present", failure.key)
	}

	require.Equal(t, workers*keysPerWorker, mapImpl.Len())

	for key := range workers * keysPerWorker {
		value, ok := mapImpl.Get(key)
		require.True(t, ok)
		require.Equal(t, key*10, value)
	}
}

type concurrentAccessFailure struct {
	err           error
	key           int
	value         int
	ok            bool
	expectedValue int
}

func entriesToMap[K comparable, V any](t *testing.T, entries []MapEntry[K, V]) map[K]V {
	t.Helper()

	result := make(map[K]V, len(entries))

	for _, entry := range entries {
		require.NotNil(t, entry)

		result[entry.Key()] = entry.Value()
	}

	return result
}
