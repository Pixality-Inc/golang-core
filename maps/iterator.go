package maps

type MapIterator[K comparable, V any] struct {
	entries []MapEntry[K, V]
	cursor  int
}

func NewMapIterator[K comparable, V any](mapImpl ReadOnlyMap[K, V]) *MapIterator[K, V] {
	return &MapIterator[K, V]{
		entries: mapImpl.Entries(),
		cursor:  0,
	}
}

func (i *MapIterator[K, V]) HasNext() bool {
	return i.cursor < len(i.entries)
}

func (i *MapIterator[K, V]) Next() MapEntry[K, V] {
	i.cursor++

	return i.entries[i.cursor-1]
}

func (i *MapIterator[K, V]) Err() error {
	return nil
}
