package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
)

const ReadBufferSize = 32 * 1024

type Closeable interface {
	Close() error
}

type Lifecycle[T Closeable] struct {
	resource    T
	hasResource bool
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	mutex       sync.Mutex
}

func NewLifecycle[T Closeable]() *Lifecycle[T] {
	return &Lifecycle[T]{
		wg:    sync.WaitGroup{},
		mutex: sync.Mutex{},
	}
}

func (l *Lifecycle[T]) Set(resource T, cancel context.CancelFunc) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.resource = resource
	l.hasResource = true
	l.cancel = cancel
}

func (l *Lifecycle[T]) Get() (T, bool) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	return l.resource, l.hasResource
}

func (l *Lifecycle[T]) Go(f func()) {
	l.wg.Go(f)
}

func (l *Lifecycle[T]) Wait() {
	l.wg.Wait()
}

func (l *Lifecycle[T]) Shutdown(ctx context.Context, resourceName string) error {
	l.mutex.Lock()

	cancel := l.cancel
	resource := l.resource
	hasResource := l.hasResource

	var zero T

	l.resource = zero
	l.hasResource = false
	l.cancel = nil

	l.mutex.Unlock()

	if cancel != nil {
		cancel()
	}

	if hasResource {
		if err := resource.Close(); err != nil && !IsClosed(ctx, err) {
			return fmt.Errorf("close %s: %w", resourceName, err)
		}
	}

	return nil
}

func IsClosed(ctx context.Context, err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, net.ErrClosed) {
		return true
	}

	return ctx.Err() != nil
}
