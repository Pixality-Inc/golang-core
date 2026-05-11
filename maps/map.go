package maps

import "github.com/pixality-inc/golang-core/iterator"

type ReadOnlyMap[K comparable, V any] interface {
	Len() int
	Entries() []MapEntry[K, V]
	HasKey(key K) bool
	Get(key K) (V, bool)
	GetEntry(key K) MapEntry[K, V]
	AsGoMap() map[K]V
	AsIterator() iterator.Iterator[MapEntry[K, V]]
}

type Map[K comparable, V any] interface {
	ReadOnlyMap[K, V]

	Set(key K, value V) error
	Delete(key K) error
}
