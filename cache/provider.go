package cache

import (
	"context"
	"errors"
	"time"
)

var (
	ErrProviderNoSuchKey = errors.New("no key found")
	ErrProviderGet       = errors.New("reading key from provider")
	ErrProviderSet       = errors.New("writing key to provider")
)

type Provider interface {
	Has(ctx context.Context, group Group, key string) (bool, error)
	Get(ctx context.Context, group Group, key string) ([]byte, error)
	Set(ctx context.Context, group Group, key string, value []byte, ttl time.Duration) error
}
