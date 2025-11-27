package http_client

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	cb "github.com/pixality-inc/golang-core/circuit_breaker"
)

// ShouldIgnoreErrorForCircuitBreaker determines if an HTTP error should be ignored
// by the circuit breaker. returns true for errors that are expected client-side results
// rather than infrastructure/availability issues.
func ShouldIgnoreErrorForCircuitBreaker(err error) bool {
	if err == nil {
		return false
	}

	// context cancellation from application level - not an infrastructure issue
	if errors.Is(err, context.Canceled) {
		return true
	}

	// check for client-side HTTP errors (4xx) - these are application logic errors
	if errors.Is(err, ErrNotFound) || errors.Is(err, ErrBadRequest) {
		return true
	}

	// check error message for status codes
	errMsg := err.Error()

	// 4xx errors are client errors, not infrastructure issues
	// 400 bad request, 401 unauthorized, 403 forbidden, 404 not found, etc.
	for statusCode := 400; statusCode < 500; statusCode++ {
		if strings.Contains(errMsg, fmt.Sprintf("non-200 http status code: %d", statusCode)) {
			return true
		}
	}

	// 5xx errors are server errors and should trigger circuit breaker
	// 500 internal server error, 502 bad gateway, 503 service unavailable, 504 gateway timeout
	for statusCode := 500; statusCode < 600; statusCode++ {
		if strings.Contains(errMsg, fmt.Sprintf("non-200 http status code: %d", statusCode)) {
			return false
		}
	}

	// network errors should trigger circuit breaker
	var netErr net.Error
	if errors.As(err, &netErr) {
		// timeout or temporary network errors trigger circuit breaker
		return false
	}

	// connection refused, no route to host, etc. - infrastructure issues
	if strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "no route to host") ||
		strings.Contains(errMsg, "network is unreachable") ||
		strings.Contains(errMsg, "i/o timeout") ||
		strings.Contains(errMsg, "connection reset") ||
		strings.Contains(errMsg, "broken pipe") {
		return false
	}

	// all other errors are considered infrastructure issues
	return false
}

// NewCircuitBreaker creates a circuit breaker configured with HTTP-specific error filtering.
func NewCircuitBreaker(config cb.Config, shouldIgnoreError func(err error) bool) cb.CircuitBreaker {
	if shouldIgnoreError == nil {
		shouldIgnoreError = ShouldIgnoreErrorForCircuitBreaker
	}

	return cb.New(config, shouldIgnoreError)
}
