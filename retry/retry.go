package retry

import (
	"context"
	"errors"
	"math"
	"net"
	"time"

	"github.com/pixality-inc/golang-core/logger"
)

// ShouldRetry retries on 5xx, 429 status codes and network errors
// does not retry on 4xx (except 429) or context errors
func ShouldRetry(statusCode int, err error) bool {
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return false
		}
	}

	if statusCode >= 500 && statusCode < 600 {
		return true
	}

	if statusCode == 429 {
		return true
	}

	if statusCode > 0 {
		return false
	}

	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return true
		}

		var opErr *net.OpError
		if errors.As(err, &opErr) {
			return true
		}

		return true
	}

	return false
}

// CalculateBackoff calculates backoff duration for retry attempt
func CalculateBackoff(attempt int, policy Policy) time.Duration {
	if policy == nil {
		return 0
	}

	backoff := float64(policy.InitialInterval()) * math.Pow(policy.BackoffCoefficient(), float64(attempt))

	if policy.MaxInterval() > 0 && time.Duration(backoff) > policy.MaxInterval() {
		return policy.MaxInterval()
	}

	return time.Duration(backoff)
}

func Do[T any](
	ctx context.Context,
	policy Policy,
	log logger.Loggable,
	operation func() (T, error),
) (T, error) {
	var zero T

	if policy == nil || policy.MaxAttempts() <= 1 {
		return operation()
	}

	logger := log.GetLogger(ctx)

	var lastResult T

	var lastErr error

	for attempt := range policy.MaxAttempts() {
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return lastResult, lastErr
			}

			return zero, ctx.Err()
		default:
		}

		result, err := operation()
		if err == nil {
			return result, nil
		}

		lastResult = result
		lastErr = err

		if attempt >= policy.MaxAttempts()-1 {
			break
		}

		backoff := CalculateBackoff(attempt, policy)

		logger.WithFields(map[string]any{
			"attempt":      attempt + 1,
			"max_attempts": policy.MaxAttempts(),
			"backoff_ms":   backoff.Milliseconds(),
		}).Warn("retrying operation after error")

		select {
		case <-ctx.Done():
			return lastResult, lastErr
		case <-time.After(backoff):
		}
	}

	return lastResult, lastErr
}

func DoWithCondition[T any](
	ctx context.Context,
	policy Policy,
	log logger.Loggable,
	operation func() (T, error),
	shouldRetry func(T, error) bool,
) (T, error) {
	var zero T

	if policy == nil || policy.MaxAttempts() <= 1 {
		return operation()
	}

	logger := log.GetLogger(ctx)

	var lastResult T

	var lastErr error

	for attempt := range policy.MaxAttempts() {
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return lastResult, lastErr
			}

			return zero, ctx.Err()
		default:
		}

		result, err := operation()

		if !shouldRetry(result, err) {
			return result, err
		}

		lastResult = result
		lastErr = err

		if attempt >= policy.MaxAttempts()-1 {
			break
		}

		backoff := CalculateBackoff(attempt, policy)

		logger.WithFields(map[string]any{
			"attempt":      attempt + 1,
			"max_attempts": policy.MaxAttempts(),
			"backoff_ms":   backoff.Milliseconds(),
		}).Warn("retrying operation after error")

		select {
		case <-ctx.Done():
			return lastResult, lastErr
		case <-time.After(backoff):
		}
	}

	return lastResult, lastErr
}
