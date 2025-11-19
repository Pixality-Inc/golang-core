package cache

import (
	"context"
)

type Key interface {
	String() string
}

type Group string

type Cache[K Key, V any] interface {
	Default() V
	Group() Group
	Has(ctx context.Context, key K) (bool, error)
	Get(ctx context.Context, key K) (V, error)
	Set(ctx context.Context, key K, value V) error
}
