package iterator

type Iterator[T any] interface {
	HasNext() bool
	Next() T
	Err() error
}

func Materialize[T any](iterator Iterator[T]) ([]T, error) {
	result := make([]T, 0)

	for iterator.HasNext() && iterator.Err() == nil {
		result = append(result, iterator.Next())
	}

	if iterator.Err() != nil {
		return result, iterator.Err()
	}

	return result, nil
}
