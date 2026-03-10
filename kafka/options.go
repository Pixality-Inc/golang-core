package kafka

import (
	"github.com/pixality-inc/golang-core/circuit_breaker"
	"github.com/pixality-inc/golang-core/retry"
)

type commonOptions struct {
	circuitBreaker circuit_breaker.CircuitBreaker
	retryPolicy    retry.Policy
}

// ProducerOption configures the producer.
type ProducerOption func(*commonOptions)

func WithProducerCircuitBreaker(cb circuit_breaker.CircuitBreaker) ProducerOption {
	return func(cfg *commonOptions) {
		cfg.circuitBreaker = cb
	}
}

func WithProducerRetryPolicy(policy retry.Policy) ProducerOption {
	return func(cfg *commonOptions) {
		cfg.retryPolicy = policy
	}
}

func applyProducerOptions(opts ...ProducerOption) *commonOptions {
	cfg := &commonOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

// ConsumerOption configures the consumer.
type ConsumerOption func(*consumerOptions)

type consumerOptions struct {
	commonOptions

	decodeErrorHandler    DecodeErrorHandler
	maxProcessingAttempts int
	failedMessageHandler  FailedMessageHandler
}

func WithConsumerCircuitBreaker(cb circuit_breaker.CircuitBreaker) ConsumerOption {
	return func(cfg *consumerOptions) {
		cfg.circuitBreaker = cb
	}
}

func WithConsumerRetryPolicy(policy retry.Policy) ConsumerOption {
	return func(cfg *consumerOptions) {
		cfg.retryPolicy = policy
	}
}

func WithDecodeErrorHandler(handler DecodeErrorHandler) ConsumerOption {
	return func(cfg *consumerOptions) {
		cfg.decodeErrorHandler = handler
	}
}

func WithMaxProcessingAttempts(n int) ConsumerOption {
	return func(cfg *consumerOptions) {
		cfg.maxProcessingAttempts = n
	}
}

func WithFailedMessageHandler(handler FailedMessageHandler) ConsumerOption {
	return func(cfg *consumerOptions) {
		cfg.failedMessageHandler = handler
	}
}

func applyConsumerOptions(opts ...ConsumerOption) *consumerOptions {
	cfg := &consumerOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}
