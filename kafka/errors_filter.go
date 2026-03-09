package kafka

import (
	"context"
	"errors"

	cb "github.com/pixality-inc/golang-core/circuit_breaker"

	"github.com/twmb/franz-go/pkg/kerr"
)

// ShouldIgnoreErrorForCircuitBreaker determines if a kafka error should be ignored
// by the circuit breaker. Returns true for errors that are expected business logic
// or configuration issues rather than infrastructure/availability problems.
func ShouldIgnoreErrorForCircuitBreaker(err error) bool {
	if err == nil {
		return false
	}

	// context cancellation from application level - not a kafka issue
	if errors.Is(err, context.Canceled) {
		return true
	}

	// SASL authentication failures are config issues, not availability
	if errors.Is(err, kerr.SaslAuthenticationFailed) {
		return true
	}

	// authorization failures are permission/config issues, not availability
	if errors.Is(err, kerr.TopicAuthorizationFailed) ||
		errors.Is(err, kerr.GroupAuthorizationFailed) ||
		errors.Is(err, kerr.ClusterAuthorizationFailed) {
		return true
	}

	return false
}

// NewCircuitBreaker creates a circuit breaker configured with kafka-specific error filtering.
func NewCircuitBreaker(config cb.Config, shouldIgnoreError func(err error) bool) cb.CircuitBreaker {
	if shouldIgnoreError == nil {
		shouldIgnoreError = ShouldIgnoreErrorForCircuitBreaker
	}

	return cb.New(config, shouldIgnoreError)
}
