package promise

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/pixality-inc/golang-core/either"
)

var (
	ErrAlreadyResolved = errors.New("already resolved")
	ErrNotResolved     = errors.New("not resolved")
	ErrChannelClosed   = errors.New("channel closed")
	ErrNilError        = errors.New("nil error")
	ErrPromiseResolved = errors.New("promise is resolved, but both value and error are nil")
	ErrFutureResolved  = ErrPromiseResolved
)

type Promise[T any] interface {
	Get(ctx context.Context) (T, error)
	Chan() <-chan either.EitherError[T]
	IsResolved() bool
	Resolve(value T) error
	Reject(err error) error
}

type Impl[T any] struct {
	value *T
	error *error

	channels []chan either.EitherError[T]
	resolved atomic.Bool
	mutex    sync.Mutex
}

func New[T any]() Promise[T] {
	return &Impl[T]{
		value:    nil,
		error:    nil,
		channels: make([]chan either.EitherError[T], 0),
		resolved: atomic.Bool{},
		mutex:    sync.Mutex{},
	}
}

func (p *Impl[T]) Get(ctx context.Context) (T, error) {
	var defaultValue T

	ch := p.addChannel()
	if value, err, ok := p.tryReadResult(ch); ok {
		return value, err
	}

	select {
	case eith, ok := <-ch:
		return p.readResult(eith, ok)

	case <-ctx.Done():
		if value, err, ok := p.tryReadResult(ch); ok {
			return value, err
		}

		if p.removeChannel(ch) {
			return defaultValue, ctx.Err()
		}

		eith, ok := <-ch

		return p.readResult(eith, ok)
	}
}

func (p *Impl[T]) Chan() <-chan either.EitherError[T] {
	return p.addChannel()
}

func (p *Impl[T]) IsResolved() bool {
	return p.resolved.Load()
}

func (p *Impl[T]) Resolve(value T) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.resolved.Load() {
		return ErrAlreadyResolved
	}

	p.value = &value

	p.complete()

	return nil
}

func (p *Impl[T]) Reject(err error) error {
	if err == nil {
		return ErrNilError
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.resolved.Load() {
		return ErrAlreadyResolved
	}

	p.error = &err

	p.complete()

	return nil
}

func (p *Impl[T]) addChannel() chan either.EitherError[T] {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	ch := make(chan either.EitherError[T], 1)

	if p.resolved.Load() {
		ch <- p.getResolvedValue(true)

		close(ch)

		return ch
	}

	p.channels = append(p.channels, ch)

	return ch
}

func (p *Impl[T]) removeChannel(ch chan either.EitherError[T]) bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.resolved.Load() {
		return false
	}

	for index, candidate := range p.channels {
		if candidate == ch {
			p.channels = append(p.channels[:index], p.channels[index+1:]...)

			close(ch)

			return true
		}
	}

	return true
}

func (p *Impl[T]) complete() {
	p.resolved.Store(true)

	result := p.getResolvedValue(false)

	for _, ch := range p.channels {
		ch <- result

		close(ch)
	}

	p.channels = nil
}

func (p *Impl[T]) tryReadResult(ch <-chan either.EitherError[T]) (T, error, bool) {
	select {
	case eith, ok := <-ch:
		value, err := p.readResult(eith, ok)

		return value, err, true
	default:
		var defaultValue T

		return defaultValue, nil, false
	}
}

func (p *Impl[T]) readResult(eith either.EitherError[T], ok bool) (T, error) {
	var defaultValue T

	if !ok {
		return defaultValue, ErrChannelClosed
	}

	value, err := eith.Value()
	if err != nil {
		return defaultValue, err
	}

	return value, nil
}

func (p *Impl[T]) getResolvedValue(checkForResolved bool) either.EitherError[T] {
	if checkForResolved && !p.IsResolved() {
		return either.Error[T](ErrNotResolved)
	}

	switch {
	case p.value != nil:
		return either.RightError[T](*p.value)
	case p.error != nil:
		return either.Error[T](*p.error)
	default:
		return either.Error[T](ErrPromiseResolved)
	}
}
