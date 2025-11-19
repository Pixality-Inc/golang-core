package redis

import (
	"context"
	"errors"
	"strings"

	cb "github.com/pixality-inc/golang-core/circuit_breaker"

	goredis "github.com/redis/go-redis/v9"
)

// ShouldIgnoreErrorForCircuitBreaker determines if a redis error should be ignored
// by the circuit breaker. returns true for errors that are expected business logic results
// rather than infrastructure/availability issues.
func ShouldIgnoreErrorForCircuitBreaker(err error) bool {
	if err == nil {
		return false
	}

	// key not found - expected business logic result (like pgx.ErrNoRows)
	if errors.Is(err, goredis.Nil) {
		return true
	}

	// context cancellation from application level - not a redis issue
	if errors.Is(err, context.Canceled) {
		return true
	}

	errMsg := err.Error()

	// WRONGTYPE - wrong data type stored in key, obv application logic error
	if strings.Contains(errMsg, "WRONGTYPE") {
		return true
	}

	// NOAUTH/WRONGPASS/NOPERM - authentication/permission errors, config issue not availability issue
	if strings.Contains(errMsg, "NOAUTH") ||
		strings.Contains(errMsg, "WRONGPASS") ||
		strings.Contains(errMsg, "NOPERM") {
		return true
	}

	// connection pool/infrastructure errors trigger circuit breaker
	if errors.Is(err, goredis.ErrClosed) {
		return false
	}

	// cluster/availability issues trigger circuit breaker
	// READONLY, MASTERDOWN, CLUSTERDOWN, LOADING
	if strings.HasPrefix(errMsg, "READONLY ") ||
		strings.HasPrefix(errMsg, "MASTERDOWN ") ||
		strings.HasPrefix(errMsg, "CLUSTERDOWN ") ||
		strings.HasPrefix(errMsg, "LOADING ") {
		return false
	}

	// max clients reached - infrastructure problem
	if strings.Contains(errMsg, "max number of clients reached") {
		return false
	}

	// all other errors are infrastructure issues and should trigger circuit breaker
	return false
}

// NewCircuitBreaker creates a circuit breaker configured with redis-specific error filtering.
func NewCircuitBreaker(config cb.Config, shouldIgnoreError func(err error) bool) cb.CircuitBreaker {
	if shouldIgnoreError == nil {
		shouldIgnoreError = ShouldIgnoreErrorForCircuitBreaker
	}

	return cb.New(config, shouldIgnoreError)
}
