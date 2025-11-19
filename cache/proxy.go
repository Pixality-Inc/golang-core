package cache

import "context"

type Proxy[K Key, V any] interface {
	Get(ctx context.Context, key K) (V, error)
}

type ProxyGetter[K Key, V any] interface {
	Get(ctx context.Context, key Key) (V, error)
}

type ProxyImpl[K Key, V any] struct {
	cache  Cache[K, V]
	getter ProxyGetter[K, V]
}

func NewProxy[K Key, V any](cache Cache[K, V], getter ProxyGetter[K, V]) Proxy[K, V] {
	return &ProxyImpl[K, V]{
		cache:  cache,
		getter: getter,
	}
}

func (p *ProxyImpl[K, V]) Get(ctx context.Context, key K) (V, error) {
	if ok, err := p.cache.Has(ctx, key); err != nil {
		return p.cache.Default(), err
	} else if ok {
		return p.cache.Get(ctx, key)
	} else {
		value, err := p.getter.Get(ctx, key)
		if err != nil {
			return p.cache.Default(), err
		}

		if err = p.cache.Set(ctx, key, value); err != nil {
			return p.cache.Default(), err
		}

		return value, nil
	}
}
