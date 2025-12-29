package redis

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/pixality-inc/golang-core/circuit_breaker"
	"github.com/pixality-inc/golang-core/json"
	"github.com/pixality-inc/golang-core/logger"

	goredis "github.com/redis/go-redis/v9"
)

//go:generate mockgen -source=redis.go -destination=mocks/redis.go -package=redis_mock Client
type Client interface {
	Close()

	SetKey(ctx context.Context, key string, value string, ttl time.Duration) error

	GetString(ctx context.Context, key string) (string, error)

	IsConnected() bool

	Subscribe(ctx context.Context, channels ...string) *PubSub
	Publish(ctx context.Context, channel string, message any) error

	SetNX(ctx context.Context, key string, value any, expiration time.Duration) (bool, error)
	Del(ctx context.Context, keys ...string) error
}

type PubSub struct {
	pubsub *goredis.PubSub
}

func (p *PubSub) Channel() <-chan *Message {
	if p == nil || p.pubsub == nil {
		return nil
	}

	return p.pubsub.Channel()
}

func (p *PubSub) Close() error {
	if p == nil || p.pubsub == nil {
		return nil
	}

	return p.pubsub.Close()
}

type Impl struct {
	log                logger.Loggable
	sentinelMasterName string
	sentinelAddresses  []string
	network            string
	protocol           int
	host               string
	port               int
	clientName         string
	username           string
	password           string
	db                 int
	mutex              sync.Mutex
	client             *goredis.Client
	circuitBreaker     circuit_breaker.CircuitBreaker
}

func NewClient(config Config, cb circuit_breaker.CircuitBreaker) Client {
	return &Impl{
		log:                logger.NewLoggableImplWithService("redis"),
		sentinelMasterName: config.SentinelMasterName(),
		sentinelAddresses:  config.SentinelAddresses(),
		network:            config.Network(),
		protocol:           config.Protocol(),
		host:               config.Host(),
		port:               config.Port(),
		clientName:         config.ClientName(),
		username:           config.Username(),
		password:           config.Password(),
		db:                 config.DB(),
		client:             nil,
		mutex:              sync.Mutex{},
		circuitBreaker:     cb,
	}
}

func (c *Impl) Close() {
	if c.client == nil {
		return
	}

	if err := c.client.Close(); err != nil {
		c.log.GetLoggerWithoutContext().WithError(err).Error("error closing redis client")
	}
}

func (c *Impl) IsConnected() bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.client != nil
}

func (c *Impl) SetKey(ctx context.Context, key string, value string, ttl time.Duration) error {
	if err := c.ensureConnected(ctx); err != nil {
		return err
	}

	c.log.GetLogger(ctx).
		WithField("key", key).
		WithField("ttl", ttl.Milliseconds()).
		Tracef("setting key %s", key)

	return circuit_breaker.Execute(c.circuitBreaker, func() error {
		return c.client.Set(ctx, key, value, ttl).Err()
	})
}

func (c *Impl) GetString(ctx context.Context, key string) (string, error) {
	cmd, err := c.getKey(ctx, key)
	if err != nil {
		return "", err
	}

	return cmd.Val(), cmd.Err()
}

func (c *Impl) Subscribe(ctx context.Context, channels ...string) *PubSub {
	if err := c.ensureConnected(ctx); err != nil {
		return nil
	}

	return &PubSub{pubsub: c.client.Subscribe(ctx, channels...)}
}

func (c *Impl) Publish(ctx context.Context, channel string, message any) error {
	if err := c.ensureConnected(ctx); err != nil {
		return err
	}

	return circuit_breaker.Execute(c.circuitBreaker, func() error {
		return c.client.Publish(ctx, channel, message).Err()
	})
}

func (c *Impl) SetNX(ctx context.Context, key string, value any, expiration time.Duration) (bool, error) {
	if err := c.ensureConnected(ctx); err != nil {
		return false, err
	}

	return circuit_breaker.ExecuteWithResult(c.circuitBreaker, func() (bool, error) {
		return c.client.SetNX(ctx, key, value, expiration).Result()
	}, false)
}

func (c *Impl) Del(ctx context.Context, keys ...string) error {
	if err := c.ensureConnected(ctx); err != nil {
		return err
	}

	return circuit_breaker.Execute(c.circuitBreaker, func() error {
		return c.client.Del(ctx, keys...).Err()
	})
}

func (c *Impl) ensureConnected(_ context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.client != nil {
		return nil
	}

	//nolint:contextcheck // connectFunc intentionally uses context.Background() for Ping
	connectFunc := func() error {
		if c.sentinelMasterName != "" {
			c.client = goredis.NewFailoverClient(&goredis.FailoverOptions{
				MasterName:    c.sentinelMasterName,
				SentinelAddrs: c.sentinelAddresses,
				Protocol:      c.protocol,
				ClientName:    c.clientName,
				Username:      c.username,
				Password:      c.password,
				DB:            c.db,
			})
		} else {
			c.client = goredis.NewClient(&goredis.Options{
				Network:    c.network,
				Protocol:   c.protocol,
				Addr:       c.host + ":" + strconv.Itoa(c.port),
				ClientName: c.clientName,
				Username:   c.username,
				Password:   c.password,
				DB:         c.db,
			})
		}

		pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := c.client.Ping(pingCtx).Err(); err != nil {
			// if ping failed clean up client and return error
			c.client.Close()
			c.client = nil

			return err
		}

		return nil
	}

	return circuit_breaker.Execute(c.circuitBreaker, connectFunc)
}

func (c *Impl) getKey(ctx context.Context, key string) (*goredis.StringCmd, error) {
	if err := c.ensureConnected(ctx); err != nil {
		return nil, err
	}

	c.log.GetLogger(ctx).
		WithField("key", key).
		Tracef("getting key %s", key)

	return circuit_breaker.ExecuteWithResult(
		c.circuitBreaker,
		func() (*goredis.StringCmd, error) {
			cmd := c.client.Get(ctx, key)

			return cmd, cmd.Err()
		},
		nil,
	)
}

func Set[T any](ctx context.Context, client Client, key string, value T, ttl time.Duration) error {
	buf, err := json.Marshal(value)
	if err != nil {
		return err
	}

	strVal := string(buf)

	return client.SetKey(ctx, key, strVal, ttl)
}

func Get[T any](ctx context.Context, client Client, key string, defaultValue T) (T, error) {
	strValue, err := client.GetString(ctx, key)
	if err != nil {
		return defaultValue, err
	}

	bytes := []byte(strValue)

	var result T

	if err = json.Unmarshal(bytes, &result); err != nil {
		return defaultValue, err
	}

	return result, nil
}

type Message = goredis.Message

func Publish[T any](ctx context.Context, client Client, channel string, message T) error {
	buf, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return client.Publish(ctx, channel, string(buf))
}

func Subscribe[T any](ctx context.Context, client Client, channels ...string) (<-chan T, func() error) {
	pubsub := client.Subscribe(ctx, channels...)
	if pubsub == nil {
		return nil, func() error { return nil }
	}

	ch := make(chan T)

	go func() {
		defer close(ch)

		for msg := range pubsub.Channel() {
			var value T
			if json.Unmarshal([]byte(msg.Payload), &value) == nil {
				ch <- value
			}
		}
	}()

	return ch, pubsub.Close
}
