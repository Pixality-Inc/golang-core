package maps

type MapEntry[K comparable, V any] interface {
	Key() K
	Value() V
}

type MapEntryImpl[K comparable, V any] struct {
	key   K
	value V
}

func NewMapEntry[K comparable, V any](key K, value V) MapEntry[K, V] {
	return &MapEntryImpl[K, V]{
		key:   key,
		value: value,
	}
}

func (e *MapEntryImpl[K, V]) Key() K {
	return e.key
}

func (e *MapEntryImpl[K, V]) Value() V {
	return e.value
}
