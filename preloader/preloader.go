package preloader

import (
	"context"
	"sync"
	"time"

	"github.com/pixality-inc/golang-core/clock"
	"github.com/pixality-inc/golang-core/errors"
	"github.com/pixality-inc/golang-core/logger"
)

var (
	ErrGetValue     = errors.New("preloader.get_value", "get value")
	ErrRefreshValue = errors.New("preloader.refresh_value", "refresh value")
)

type LoaderFunction[T any] = func(ctx context.Context) (T, error)

type Config interface {
	Name() string
	TTL() time.Duration
}

type Preloader[T any] interface {
	Name() string
	DefaultValue() T
	TTL() time.Duration
	LastRefreshAt() time.Time
	Value(ctx context.Context) (T, error)
	Refresh(ctx context.Context) (T, error)
}

type Impl[T any] struct {
	log           logger.Loggable
	name          string
	defaultValue  T
	ttl           time.Duration
	loader        LoaderFunction[T]
	currentValue  T
	lastRefreshAt time.Time
	stateMutex    sync.RWMutex // guards currentValue and lastRefreshAt
	mutex         sync.Mutex   // serializes loader calls
}

func New[T any](
	config Config,
	defaultValue T,
	loader LoaderFunction[T],
) *Impl[T] {
	return &Impl[T]{
		log: logger.NewLoggableImplWithServiceAndFields(
			"preloader",
			logger.Fields{
				"name": config.Name(),
			},
		),
		name:          config.Name(),
		defaultValue:  defaultValue,
		ttl:           config.TTL(),
		loader:        loader,
		currentValue:  defaultValue,
		lastRefreshAt: time.Time{},
		stateMutex:    sync.RWMutex{},
		mutex:         sync.Mutex{},
	}
}

func (p *Impl[T]) Name() string {
	return p.name
}

func (p *Impl[T]) DefaultValue() T {
	return p.defaultValue
}

func (p *Impl[T]) TTL() time.Duration {
	return p.ttl
}

func (p *Impl[T]) LastRefreshAt() time.Time {
	p.stateMutex.RLock()
	defer p.stateMutex.RUnlock()

	return p.lastRefreshAt
}

func (p *Impl[T]) Value(ctx context.Context) (T, error) {
	if value, ok := p.freshValue(ctx); ok {
		return value, nil
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	// another caller may have refreshed the value while this one waited for the lock
	if value, ok := p.freshValue(ctx); ok {
		return value, nil
	}

	newValue, err := p.load(ctx)
	if err != nil {
		return p.defaultValue, errors.Join(ErrGetValue, errors.Join(ErrRefreshValue, err))
	}

	return newValue, nil
}

func (p *Impl[T]) Refresh(ctx context.Context) (T, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	newValue, err := p.load(ctx)
	if err != nil {
		return p.defaultValue, errors.Join(ErrRefreshValue, err)
	}

	return newValue, nil
}

// load must be called with p.mutex held
func (p *Impl[T]) load(ctx context.Context) (T, error) {
	newValue, err := p.loader(ctx)
	if err != nil {
		return p.defaultValue, err
	}

	p.stateMutex.Lock()
	defer p.stateMutex.Unlock()

	p.currentValue = newValue
	p.lastRefreshAt = clock.GetClock(ctx).Now()

	return newValue, nil
}

func (p *Impl[T]) freshValue(ctx context.Context) (T, bool) {
	now := clock.GetClock(ctx).Now()

	p.stateMutex.RLock()
	defer p.stateMutex.RUnlock()

	if now.After(p.lastRefreshAt.Add(p.ttl)) {
		return p.defaultValue, false
	}

	return p.currentValue, true
}
