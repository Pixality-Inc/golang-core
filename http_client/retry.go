package http_client

import (
	"context"
	"errors"
	"math"
	"net"
	"time"
)

func shouldRetry(statusCode int, err error) bool {
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

		if errors.As(err, &netErr) && netErr.Temporary() {
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

func calculateBackoff(attempt int, policy *RetryPolicy) time.Duration {
	if policy == nil {
		return 0
	}

	backoff := float64(policy.InitialInterval) * math.Pow(policy.BackoffCoefficient, float64(attempt))

	if policy.MaxInterval > 0 && time.Duration(backoff) > policy.MaxInterval {
		return policy.MaxInterval
	}

	return time.Duration(backoff)
}

func (c *ClientImpl) doWithRetry(
	ctx context.Context,
	operation func() (*Response, error),
) (*Response, error) {
	policy := c.config.RetryPolicy()
	if policy == nil || policy.MaxAttempts <= 1 {
		return operation()
	}

	log := c.log.GetLogger(ctx)

	var lastResponse *Response
	var lastErr error

	for attempt := 0; attempt < policy.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return lastResponse, lastErr
			}
			return nil, ctx.Err()
		default:
		}

		response, err := operation()

		if err == nil && (response == nil || !shouldRetry(response.StatusCode, nil)) {
			return response, nil
		}

		statusCode := 0
		if response != nil {
			statusCode = response.StatusCode
		}

		if !shouldRetry(statusCode, err) {
			return response, err
		}

		lastResponse = response
		lastErr = err

		if attempt >= policy.MaxAttempts-1 {
			break
		}

		backoff := calculateBackoff(attempt, policy)

		log.WithFields(map[string]any{
			"attempt":      attempt + 1,
			"max_attempts": policy.MaxAttempts,
			"backoff_ms":   backoff.Milliseconds(),
			"status_code":  statusCode,
		}).Warn("retrying request after error")

		select {
		case <-ctx.Done():
			return lastResponse, lastErr
		case <-time.After(backoff):
		}
	}

	return lastResponse, lastErr
}
