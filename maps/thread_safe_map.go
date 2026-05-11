package maps

import (
	"maps"
	"sync"

	"github.com/pixality-inc/golang-core/iterator"
)

type ThreadSafeMapImpl[K comparable, V any] struct {
	mutex       sync.RWMutex
	internalMap map[K]V
}

func NewThreadSafeMap[K comparable, V any](initialValue map[K]V) Map[K, V] {
	var internalMap map[K]V

	if initialValue != nil {
		internalMap = make(map[K]V, len(initialValue))

		maps.Copy(internalMap, initialValue)
	} else {
		internalMap = make(map[K]V)
	}

	return &ThreadSafeMapImpl[K, V]{
		mutex:       sync.RWMutex{},
		internalMap: internalMap,
	}
}

func (m *ThreadSafeMapImpl[K, V]) Len() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return len(m.internalMap)
}

func (m *ThreadSafeMapImpl[K, V]) Entries() []MapEntry[K, V] {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	entries := make([]MapEntry[K, V], 0, len(m.internalMap))

	for key, value := range m.internalMap {
		entries = append(entries, NewMapEntry(key, value))
	}

	return entries
}

func (m *ThreadSafeMapImpl[K, V]) HasKey(key K) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	_, ok := m.internalMap[key]

	return ok
}

func (m *ThreadSafeMapImpl[K, V]) Get(key K) (V, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	value, ok := m.internalMap[key]

	return value, ok
}

func (m *ThreadSafeMapImpl[K, V]) GetEntry(key K) MapEntry[K, V] {
	value, ok := m.Get(key)
	if !ok {
		return nil
	}

	return NewMapEntry(key, value)
}

func (m *ThreadSafeMapImpl[K, V]) AsGoMap() map[K]V {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	mapCopy := make(map[K]V, len(m.internalMap))

	maps.Copy(mapCopy, m.internalMap)

	return mapCopy
}

func (m *ThreadSafeMapImpl[K, V]) AsIterator() iterator.Iterator[MapEntry[K, V]] {
	return NewMapIterator(m)
}

func (m *ThreadSafeMapImpl[K, V]) Set(key K, value V) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.internalMap[key] = value

	return nil
}

func (m *ThreadSafeMapImpl[K, V]) Delete(key K) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.internalMap, key)

	return nil
}
