package provider

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pixality-inc/golang-core/cache"
)

type Entry struct {
	Value     []byte
	ExpiresAt time.Time
}

func NewEntry(value []byte, expiresAt time.Time) Entry {
	return Entry{
		Value:     value,
		ExpiresAt: expiresAt,
	}
}

type Memory struct {
	storage map[string]Entry
	mutex   sync.RWMutex
}

func NewMemory() *Memory {
	return &Memory{
		storage: make(map[string]Entry),
		mutex:   sync.RWMutex{},
	}
}

func (p *Memory) Has(
	_ context.Context,
	group cache.Group,
	key string,
) (bool, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	_, ok := p.storage[p.key(group, key)]

	return ok, nil
}

func (p *Memory) Get(
	_ context.Context,
	group cache.Group,
	key string,
) ([]byte, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	groupKey := p.key(group, key)

	if value, ok := p.storage[groupKey]; ok {
		if time.Now().After(value.ExpiresAt) {
			delete(p.storage, groupKey)

			return nil, fmt.Errorf("%w: %s", cache.ErrProviderNoSuchKey, groupKey)
		}

		return value.Value, nil
	} else {
		return nil, fmt.Errorf("%w: %s", cache.ErrProviderNoSuchKey, groupKey)
	}
}

func (p *Memory) Set(
	_ context.Context,
	group cache.Group,
	key string,
	value []byte,
	ttl time.Duration,
) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	groupKey := p.key(group, key)

	p.storage[groupKey] = NewEntry(value, time.Now().Add(ttl))

	return nil
}

func (p *Memory) key(group cache.Group, key string) string {
	return string(group) + ":" + key
}
