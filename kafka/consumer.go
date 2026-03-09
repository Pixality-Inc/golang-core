package kafka

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pixality-inc/golang-core/circuit_breaker"
	"github.com/pixality-inc/golang-core/logger"
	"github.com/pixality-inc/golang-core/retry"

	"github.com/twmb/franz-go/pkg/kgo"
)

type processAction int

const (
	actionRetry processAction = iota
	actionStop
	actionSkip
)

type Handler[T any] func(ctx context.Context, msg Message[T]) error

// DecodeErrorHandler is called when a consumed message cannot be decoded.
// Return nil to skip the message (and auto-commit if enabled).
// Return a non-nil error to stop processing the partition.
type DecodeErrorHandler func(ctx context.Context, topic string, partition int32, offset int64, err error) error

type Consumer[T any] interface {
	Consume(ctx context.Context, handler Handler[T]) error
	IsConnected() bool
	Ping(ctx context.Context) error
	Stop() error
}

// FailedMessageHandler is called when a message exhausts all processing attempts.
// Receives the raw bytes and metadata. Return nil to commit and skip the message,
// return a non-nil error to stop processing the partition.
type FailedMessageHandler func(ctx context.Context, topic string, partition int32, offset int64, value []byte, err error) error

type topicPartitionOffset struct {
	topic     string
	partition int32
	offset    int64
}

type consumerImpl[T any] struct {
	log                   logger.Loggable
	config                ConsumerConfig
	protocol              Protocol[T]
	client                *kgo.Client
	mutex                 sync.RWMutex
	circuitBreaker        circuit_breaker.CircuitBreaker
	retryPolicy           retry.Policy
	decodeErrorHandler    DecodeErrorHandler
	maxProcessingAttempts int
	failedMessageHandler  FailedMessageHandler
	attempts              map[topicPartitionOffset]int
}

func NewConsumer[T any](config ConsumerConfig, protocol Protocol[T], opts ...Option) (Consumer[T], error) {
	if err := validateConsumerConfig(config); err != nil {
		return nil, err
	}

	options := applyOptions(opts...)

	var cb circuit_breaker.CircuitBreaker
	if options.circuitBreaker != nil {
		cb = options.circuitBreaker
	} else if config.CircuitBreaker() != nil {
		cb = NewCircuitBreaker(config.CircuitBreaker(), nil)
	}

	retryPolicy := config.RetryPolicy()
	if options.retryPolicy != nil {
		retryPolicy = options.retryPolicy
	}

	maxAttempts := config.MaxProcessingAttempts()
	if options.maxProcessingAttempts > 0 {
		maxAttempts = options.maxProcessingAttempts
	}

	return &consumerImpl[T]{
		log: logger.NewLoggableImplWithServiceAndFields("kafka_consumer", logger.Fields{
			"topic": config.Topic(),
		}),
		config:                config,
		protocol:              protocol,
		circuitBreaker:        cb,
		retryPolicy:           retryPolicy,
		decodeErrorHandler:    options.decodeErrorHandler,
		maxProcessingAttempts: maxAttempts,
		failedMessageHandler:  options.failedMessageHandler,
		attempts:              make(map[topicPartitionOffset]int),
	}, nil
}

func (c *consumerImpl[T]) ensureConnected(ctx context.Context) (*kgo.Client, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.client != nil {
		return c.client, nil
	}

	connectFunc := func() error {
		kgoOpts, err := buildKgoOpts(c.config)
		if err != nil {
			return err
		}

		kgoOpts = append(kgoOpts,
			kgo.ConsumeTopics(c.config.Topic()),
			kgo.ConsumerGroup(c.config.GroupID()),
			kgo.DisableAutoCommit(),
		)

		client, err := kgo.NewClient(kgoOpts...)
		if err != nil {
			return fmt.Errorf("failed to create kafka consumer client: %w", err)
		}

		pingCtx, cancel := context.WithTimeout(ctx, c.config.ConnectTimeout())
		defer cancel()

		if err = client.Ping(pingCtx); err != nil {
			client.Close()

			return fmt.Errorf("failed to ping kafka broker: %w", err)
		}

		c.client = client

		return nil
	}

	if err := circuit_breaker.Execute(c.circuitBreaker, connectFunc); err != nil {
		return nil, err
	}

	return c.client, nil
}

func (c *consumerImpl[T]) Consume(ctx context.Context, handler Handler[T]) error {
	client, err := c.ensureConnected(ctx)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			c.flushOffsets(ctx, client)

			return ctx.Err()
		default:
		}

		fetches := client.PollFetches(ctx)

		if ctx.Err() != nil {
			c.flushOffsets(ctx, client)

			return ctx.Err()
		}

		fetches.EachError(func(topic string, partition int32, err error) {
			c.log.GetLogger(ctx).
				WithError(err).
				WithField("topic", topic).
				WithField("partition", partition).
				Warn("fetch error")
		})

		type topicPartition struct {
			topic     string
			partition int32
		}

		failed := make(map[topicPartition]bool)

		fetches.EachRecord(func(record *kgo.Record) {
			topicPart := topicPartition{topic: record.Topic, partition: record.Partition}
			if failed[topicPart] {
				return
			}

			if c.processRecord(ctx, record, handler) {
				failed[topicPart] = true
			}
		})
	}
}

func (c *consumerImpl[T]) flushOffsets(ctx context.Context, client *kgo.Client) {
	if !c.config.AutoCommit() {
		return
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), c.config.ConnectTimeout())
	defer cancel()

	//nolint:contextcheck // intentionally using context.Background() because the parent context is already canceled
	if err := client.CommitUncommittedOffsets(shutdownCtx); err != nil {
		c.log.GetLogger(ctx).WithError(err).Error("failed to commit offsets on shutdown")
	}
}

func (c *consumerImpl[T]) getClient() (*kgo.Client, error) {
	c.mutex.RLock()
	client := c.client
	c.mutex.RUnlock()

	if client == nil {
		return nil, ErrNotConnected
	}

	return client, nil
}

func (c *consumerImpl[T]) commitRecord(ctx context.Context, record *kgo.Record) error {
	cl, err := c.getClient()
	if err != nil {
		return err
	}

	return cl.CommitRecords(ctx, record)
}

// processRecord processes a single record. Returns true if processing of subsequent
// records in this partition should stop (to prevent auto-committing past a failed message).
func (c *consumerImpl[T]) processRecord(
	ctx context.Context,
	record *kgo.Record,
	handler Handler[T],
) bool {
	value, err := c.protocol.Decode(ctx, record.Value)
	if err != nil {
		if c.decodeErrorHandler != nil {
			if handlerErr := c.decodeErrorHandler(ctx, record.Topic, record.Partition, record.Offset, err); handlerErr != nil {
				c.log.GetLogger(ctx).
					WithError(handlerErr).
					WithField("topic", record.Topic).
					WithField("partition", record.Partition).
					WithField("offset", record.Offset).
					Error("decode error handler rejected message")

				return true
			}
		} else {
			c.log.GetLogger(ctx).
				WithError(err).
				WithField("topic", record.Topic).
				WithField("partition", record.Partition).
				WithField("offset", record.Offset).
				Error("failed to decode message, skipping")
		}

		if c.config.AutoCommit() {
			if commitErr := c.commitRecord(ctx, record); commitErr != nil {
				c.log.GetLogger(ctx).
					WithError(commitErr).
					WithField("topic", record.Topic).
					WithField("partition", record.Partition).
					WithField("offset", record.Offset).
					Error("failed to auto-commit skipped record")
			}
		}

		return false
	}

	msg := Message[T]{
		Value:     value,
		Key:       record.Key,
		Topic:     record.Topic,
		Partition: record.Partition,
		Offset:    record.Offset,
		Timestamp: record.Timestamp,
		Headers:   convertFromKgoHeaders(record.Headers),
		Commit: func(commitCtx context.Context) error {
			return c.commitRecord(commitCtx, record)
		},
	}

	tpo := topicPartitionOffset{
		topic:     record.Topic,
		partition: record.Partition,
		offset:    record.Offset,
	}

	for {
		_, handlerErr := retry.Do(ctx, c.retryPolicy, c.log, func() (struct{}, error) {
			return struct{}{}, handler(ctx, msg)
		})
		if handlerErr == nil {
			break
		}

		c.log.GetLogger(ctx).
			WithError(handlerErr).
			WithField("topic", record.Topic).
			WithField("partition", record.Partition).
			WithField("offset", record.Offset).
			Error("handler failed after retries")

		switch c.handleFailedRecord(ctx, record, tpo, handlerErr) {
		case actionStop:
			return true
		case actionSkip:
			return false
		case actionRetry:
			if ctx.Err() != nil {
				return true
			}

			if c.retryPolicy == nil {
				select {
				case <-ctx.Done():
					return true
				case <-time.After(time.Second):
				}
			}
		}
	}

	delete(c.attempts, tpo)

	if c.config.AutoCommit() {
		if commitErr := c.commitRecord(ctx, record); commitErr != nil {
			c.log.GetLogger(ctx).
				WithError(commitErr).
				WithField("topic", record.Topic).
				WithField("partition", record.Partition).
				WithField("offset", record.Offset).
				Error("failed to auto-commit record")
		}
	}

	return false
}

// handleFailedRecord handles a record that failed processing after retries.
// Returns actionRetry to retry, actionStop to stop the partition, actionSkip to commit and move on.
func (c *consumerImpl[T]) handleFailedRecord(
	ctx context.Context,
	record *kgo.Record,
	tpo topicPartitionOffset,
	handlerErr error,
) processAction {
	if c.maxProcessingAttempts <= 0 {
		return actionRetry
	}

	c.attempts[tpo]++

	if c.attempts[tpo] < c.maxProcessingAttempts {
		return actionRetry
	}

	defer delete(c.attempts, tpo)

	if c.failedMessageHandler != nil {
		if fErr := c.failedMessageHandler(ctx, record.Topic, record.Partition, record.Offset, record.Value, handlerErr); fErr != nil {
			c.log.GetLogger(ctx).
				WithError(fErr).
				WithField("topic", record.Topic).
				WithField("partition", record.Partition).
				WithField("offset", record.Offset).
				Error("failed message handler returned error")

			return actionStop
		}
	} else {
		c.log.GetLogger(ctx).
			WithField("topic", record.Topic).
			WithField("partition", record.Partition).
			WithField("offset", record.Offset).
			Warn("message exhausted all processing attempts, skipping")
	}

	if c.config.AutoCommit() {
		if commitErr := c.commitRecord(ctx, record); commitErr != nil {
			c.log.GetLogger(ctx).
				WithError(commitErr).
				WithField("topic", record.Topic).
				WithField("partition", record.Partition).
				WithField("offset", record.Offset).
				Error("failed to auto-commit exhausted record")
		}
	}

	return actionSkip
}

func (c *consumerImpl[T]) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.client != nil
}

func (c *consumerImpl[T]) Ping(ctx context.Context) error {
	c.mutex.RLock()
	client := c.client
	c.mutex.RUnlock()

	if client == nil {
		return ErrNotConnected
	}

	return client.Ping(ctx)
}

func (c *consumerImpl[T]) Stop() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.client != nil {
		c.client.Close()
		c.client = nil
	}

	return nil
}
