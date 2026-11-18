package redis

import (
	"context"
	"errors"
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

	// HSet sets the field of the hash and reports whether the field is new
	HSet(ctx context.Context, key string, field string, value any) (bool, error)
	// HSetNX sets the field of the hash only when it is absent and reports whether it was set
	HSetNX(ctx context.Context, key string, field string, value any) (bool, error)
	// HIncrBy increments the integer field of the hash and returns the new value
	HIncrBy(ctx context.Context, key string, field string, increment int64) (int64, error)
	HGetAll(ctx context.Context, key string) (map[string]string, error)
	HDel(ctx context.Context, key string, fields ...string) error

	// PExpire sets the ttl of the key and reports whether the key exists
	PExpire(ctx context.Context, key string, ttl time.Duration) (bool, error)
}

var ErrPubSubNotConnected = errors.New("redis subscription is not connected")

// ChannelOption tunes the go channel of a subscription: WithChannelSize, WithChannelSendTimeout
type ChannelOption = goredis.ChannelOption

var (
	// WithChannelSize sets the buffer of the go channel of a subscription (100 by default)
	WithChannelSize = goredis.WithChannelSize
	// WithChannelSendTimeout sets how long a message waits for a full channel before it is
	// dropped (a minute by default)
	WithChannelSendTimeout = goredis.WithChannelSendTimeout
)

type PubSub struct {
	pubsub         *goredis.PubSub
	circuitBreaker circuit_breaker.CircuitBreaker
}

func (p *PubSub) Channel() <-chan *Message {
	if p == nil || p.pubsub == nil {
		return nil
	}

	return p.pubsub.Channel()
}

func (p *PubSub) ChannelWithOptions(opts ...ChannelOption) <-chan *Message {
	if p == nil || p.pubsub == nil {
		return nil
	}

	return p.pubsub.Channel(opts...)
}

// Subscribe adds channels to the subscription
func (p *PubSub) Subscribe(ctx context.Context, channels ...string) error {
	if p == nil || p.pubsub == nil {
		return ErrPubSubNotConnected
	}

	return circuit_breaker.Execute(p.circuitBreaker, func() error {
		return p.pubsub.Subscribe(ctx, channels...)
	})
}

// Unsubscribe removes channels from the subscription, the subscription itself stays open
func (p *PubSub) Unsubscribe(ctx context.Context, channels ...string) error {
	if p == nil || p.pubsub == nil {
		return ErrPubSubNotConnected
	}

	return circuit_breaker.Execute(p.circuitBreaker, func() error {
		return p.pubsub.Unsubscribe(ctx, channels...)
	})
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

	return &PubSub{
		pubsub:         c.client.Subscribe(ctx, channels...),
		circuitBreaker: c.circuitBreaker,
	}
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

func (c *Impl) HSet(ctx context.Context, key string, field string, value any) (bool, error) {
	if err := c.ensureConnected(ctx); err != nil {
		return false, err
	}

	return circuit_breaker.ExecuteWithResult(c.circuitBreaker, func() (bool, error) {
		added, err := c.client.HSet(ctx, key, field, value).Result()

		return added > 0, err
	}, false)
}

func (c *Impl) HSetNX(ctx context.Context, key string, field string, value any) (bool, error) {
	if err := c.ensureConnected(ctx); err != nil {
		return false, err
	}

	return circuit_breaker.ExecuteWithResult(c.circuitBreaker, func() (bool, error) {
		return c.client.HSetNX(ctx, key, field, value).Result()
	}, false)
}

func (c *Impl) HIncrBy(ctx context.Context, key string, field string, increment int64) (int64, error) {
	if err := c.ensureConnected(ctx); err != nil {
		return 0, err
	}

	return circuit_breaker.ExecuteWithResult(c.circuitBreaker, func() (int64, error) {
		return c.client.HIncrBy(ctx, key, field, increment).Result()
	}, 0)
}

func (c *Impl) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	if err := c.ensureConnected(ctx); err != nil {
		return nil, err
	}

	return circuit_breaker.ExecuteWithResult(c.circuitBreaker, func() (map[string]string, error) {
		return c.client.HGetAll(ctx, key).Result()
	}, nil)
}

func (c *Impl) HDel(ctx context.Context, key string, fields ...string) error {
	if err := c.ensureConnected(ctx); err != nil {
		return err
	}

	return circuit_breaker.Execute(c.circuitBreaker, func() error {
		return c.client.HDel(ctx, key, fields...).Err()
	})
}

func (c *Impl) PExpire(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if err := c.ensureConnected(ctx); err != nil {
		return false, err
	}

	return circuit_breaker.ExecuteWithResult(c.circuitBreaker, func() (bool, error) {
		return c.client.PExpire(ctx, key, ttl).Result()
	}, false)
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
