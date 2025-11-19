package cache

import (
	"context"
	"errors"
	"time"
)

type Impl[K Key, V any] struct {
	group        Group
	marshaller   Marshaller
	provider     Provider
	defaultValue V
	ttl          time.Duration
}

func NewCache[K Key, V any](
	group Group,
	marshaller Marshaller,
	provider Provider,
	defaultValue V,
	ttl time.Duration,
) Cache[K, V] {
	return &Impl[K, V]{
		group:        group,
		marshaller:   marshaller,
		provider:     provider,
		defaultValue: defaultValue,
		ttl:          ttl,
	}
}

func (c *Impl[K, V]) Default() V {
	return c.defaultValue
}

func (c *Impl[K, V]) Group() Group {
	return c.group
}

func (c *Impl[K, V]) Has(ctx context.Context, key K) (bool, error) {
	return c.provider.Has(ctx, c.group, key.String())
}

func (c *Impl[K, V]) Get(ctx context.Context, key K) (V, error) {
	valueBytes, err := c.provider.Get(ctx, c.group, key.String())
	switch {
	case errors.Is(err, ErrProviderNoSuchKey):
		return c.defaultValue, nil
	case err != nil:
		return c.defaultValue, errors.Join(ErrProviderGet, err)
	}

	var result V

	if err = c.marshaller.Unmarshal(valueBytes, &result); err != nil {
		return c.defaultValue, errors.Join(ErrUnmarshal, err)
	}

	return result, nil
}

func (c *Impl[K, V]) Set(ctx context.Context, key K, value V) error {
	valueBytes, err := c.marshaller.Marshal(value)
	if err != nil {
		return errors.Join(ErrMarshal, err)
	}

	if err = c.provider.Set(ctx, c.group, key.String(), valueBytes, c.ttl); err != nil {
		return errors.Join(ErrProviderSet, err)
	}

	return nil
}
