package iterator

import (
	"errors"
	"fmt"
)

var (
	ErrNotEnoughItems = errors.New("not enough items")
	ErrNegativeCount  = errors.New("negative count")
	ErrNilIterator    = errors.New("nil iterator")
)

type Peekable[T any] interface {
	Iterator[T]

	Peek() (T, error)
	Peek2() (T, T, error)
	Peek3() (T, T, T, error)
	PeekN(num int) ([]T, error)
}

type Consumable[T any] interface {
	Iterator[T]

	Consume() error
	Consume2() error
	Consume3() error
	ConsumeN(num int) error
}

type PeekableConsumable[T any] interface {
	Peekable[T]
	Consumable[T]
}

type PeekableConsumableImpl[T any] struct {
	iterator Iterator[T]
	buffer   []T
	err      error
}

func NewPeekableConsumable[T any](iterator Iterator[T]) PeekableConsumable[T] {
	return &PeekableConsumableImpl[T]{
		iterator: iterator,
	}
}

func (i *PeekableConsumableImpl[T]) HasNext() bool {
	return i.iterator.HasNext()
}

func (i *PeekableConsumableImpl[T]) Next() T {
	return i.iterator.Next()
}

func (i *PeekableConsumableImpl[T]) Peek() (T, error) {
	values, err := i.PeekN(1)
	if err != nil {
		var zero T

		return zero, err
	}

	return values[0], nil
}

func (i *PeekableConsumableImpl[T]) Peek2() (T, T, error) {
	values, err := i.PeekN(2)
	if err != nil {
		var zero T

		return zero, zero, err
	}

	return values[0], values[1], nil
}

func (i *PeekableConsumableImpl[T]) Peek3() (T, T, T, error) {
	values, err := i.PeekN(3)
	if err != nil {
		var zero T

		return zero, zero, zero, err
	}

	return values[0], values[1], values[2], nil
}

func (i *PeekableConsumableImpl[T]) PeekN(num int) ([]T, error) {
	if err := i.fillBuffer(num); err != nil {
		return nil, err
	}

	values := make([]T, num)
	copy(values, i.buffer[:num])

	return values, nil
}

func (i *PeekableConsumableImpl[T]) Consume() error {
	return i.ConsumeN(1)
}

func (i *PeekableConsumableImpl[T]) Consume2() error {
	return i.ConsumeN(2)
}

func (i *PeekableConsumableImpl[T]) Consume3() error {
	return i.ConsumeN(3)
}

func (i *PeekableConsumableImpl[T]) ConsumeN(num int) error {
	if err := i.fillBuffer(num); err != nil {
		return err
	}

	var zero T
	for idx := range num {
		i.buffer[idx] = zero
	}

	i.buffer = i.buffer[num:]

	return nil
}

func (i *PeekableConsumableImpl[T]) Err() error {
	return i.err
}

func (i *PeekableConsumableImpl[T]) fillBuffer(num int) error {
	if num < 0 {
		return fmt.Errorf("%w: %d", ErrNegativeCount, num)
	}

	if num == 0 {
		return nil
	}

	if i.err != nil {
		return i.err
	}

	if i.iterator == nil {
		return ErrNilIterator
	}

	for len(i.buffer) < num {
		if !i.iterator.HasNext() {
			if err := i.iterator.Err(); err != nil {
				i.err = err

				return err
			}

			return fmt.Errorf("%w: requested %d, available %d", ErrNotEnoughItems, num, len(i.buffer))
		}

		i.buffer = append(i.buffer, i.iterator.Next())

		if err := i.iterator.Err(); err != nil {
			i.err = err

			return err
		}
	}

	return nil
}
