package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/pixality-inc/golang-core/cache"
	"github.com/pixality-inc/golang-core/redis"

	goredis "github.com/redis/go-redis/v9"
)

type Redis struct {
	client redis.Client
}

func NewRedis(client redis.Client) *Redis {
	return &Redis{
		client: client,
	}
}

func (p *Redis) Has(
	ctx context.Context,
	group cache.Group,
	key string,
) (bool, error) {
	_, err := p.client.GetString(ctx, p.key(group, key))
	switch {
	case errors.Is(err, goredis.Nil):
		return false, nil
	case err != nil:
		return false, err
	default:
		return true, nil
	}
}

func (p *Redis) Get(
	ctx context.Context,
	group cache.Group,
	key string,
) ([]byte, error) {
	groupKey := p.key(group, key)

	str, err := p.client.GetString(ctx, groupKey)
	switch {
	case errors.Is(err, goredis.Nil):
		return nil, fmt.Errorf("%w: %s", cache.ErrProviderNoSuchKey, groupKey)
	case err != nil:
		return nil, err
	}

	return []byte(str), nil
}

func (p *Redis) Set(
	ctx context.Context,
	group cache.Group,
	key string,
	value []byte,
	ttl time.Duration,
) error {
	if err := p.client.SetKey(ctx, p.key(group, key), string(value), ttl); err != nil {
		return err
	}

	return nil
}

func (p *Redis) key(group cache.Group, key string) string {
	return string(group) + ":" + key
}
