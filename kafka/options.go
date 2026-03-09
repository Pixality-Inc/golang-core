package kafka

import (
	"github.com/pixality-inc/golang-core/circuit_breaker"
	"github.com/pixality-inc/golang-core/retry"
)

type optionsConfig struct {
	circuitBreaker        circuit_breaker.CircuitBreaker
	retryPolicy           retry.Policy
	decodeErrorHandler    DecodeErrorHandler
	maxProcessingAttempts int
	failedMessageHandler  FailedMessageHandler
}

type Option func(*optionsConfig)

func WithCircuitBreaker(cb circuit_breaker.CircuitBreaker) Option {
	return func(cfg *optionsConfig) {
		cfg.circuitBreaker = cb
	}
}

func WithRetryPolicy(policy retry.Policy) Option {
	return func(cfg *optionsConfig) {
		cfg.retryPolicy = policy
	}
}

func WithDecodeErrorHandler(handler DecodeErrorHandler) Option {
	return func(cfg *optionsConfig) {
		cfg.decodeErrorHandler = handler
	}
}

func WithMaxProcessingAttempts(n int) Option {
	return func(cfg *optionsConfig) {
		cfg.maxProcessingAttempts = n
	}
}

func WithFailedMessageHandler(handler FailedMessageHandler) Option {
	return func(cfg *optionsConfig) {
		cfg.failedMessageHandler = handler
	}
}

func applyOptions(opts ...Option) *optionsConfig {
	cfg := &optionsConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}
