package kafka

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pixality-inc/golang-core/retry"
)

type stubCircuitBreaker struct{}

func (s *stubCircuitBreaker) Execute(fn func() error) error                         { return fn() }
func (s *stubCircuitBreaker) ExecuteWithResult(fn func() (any, error)) (any, error) { return fn() }

func TestApplyConsumerOptions(t *testing.T) {
	t.Parallel()

	t.Run("empty options", func(t *testing.T) {
		t.Parallel()

		cfg := applyConsumerOptions()
		require.Nil(t, cfg.circuitBreaker)
		require.Nil(t, cfg.retryPolicy)
		require.Nil(t, cfg.decodeErrorHandler)
		require.Zero(t, cfg.maxProcessingAttempts)
		require.Nil(t, cfg.failedMessageHandler)
	})

	t.Run("WithConsumerCircuitBreaker", func(t *testing.T) {
		t.Parallel()

		cb := &stubCircuitBreaker{}
		cfg := applyConsumerOptions(WithConsumerCircuitBreaker(cb))
		require.Same(t, cb, cfg.circuitBreaker)
	})

	t.Run("WithConsumerRetryPolicy", func(t *testing.T) {
		t.Parallel()

		policy := &retry.ConfigYaml{EnabledValue: true, MaxAttemptsValue: 3}
		cfg := applyConsumerOptions(WithConsumerRetryPolicy(policy))
		require.Same(t, policy, cfg.retryPolicy)
	})

	t.Run("WithDecodeErrorHandler", func(t *testing.T) {
		t.Parallel()

		handler := func(_ context.Context, _ string, _ int32, _ int64, _ error) error { return nil }
		cfg := applyConsumerOptions(WithDecodeErrorHandler(handler))
		require.NotNil(t, cfg.decodeErrorHandler)
	})

	t.Run("WithMaxProcessingAttempts", func(t *testing.T) {
		t.Parallel()

		cfg := applyConsumerOptions(WithMaxProcessingAttempts(5))
		require.Equal(t, 5, cfg.maxProcessingAttempts)
	})

	t.Run("WithFailedMessageHandler", func(t *testing.T) {
		t.Parallel()

		handler := func(_ context.Context, _ string, _ int32, _ int64, _ []byte, _ error) error { return nil }
		cfg := applyConsumerOptions(WithFailedMessageHandler(handler))
		require.NotNil(t, cfg.failedMessageHandler)
	})
}

func TestApplyProducerOptions(t *testing.T) {
	t.Parallel()

	t.Run("empty options", func(t *testing.T) {
		t.Parallel()

		cfg := applyProducerOptions()
		require.Nil(t, cfg.circuitBreaker)
		require.Nil(t, cfg.retryPolicy)
	})

	t.Run("WithProducerCircuitBreaker", func(t *testing.T) {
		t.Parallel()

		cb := &stubCircuitBreaker{}
		cfg := applyProducerOptions(WithProducerCircuitBreaker(cb))
		require.Same(t, cb, cfg.circuitBreaker)
	})

	t.Run("WithProducerRetryPolicy", func(t *testing.T) {
		t.Parallel()

		policy := &retry.ConfigYaml{EnabledValue: true, MaxAttemptsValue: 3}
		cfg := applyProducerOptions(WithProducerRetryPolicy(policy))
		require.Same(t, policy, cfg.retryPolicy)
	})
}
