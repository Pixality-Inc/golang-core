package circuit_breaker

import (
	"errors"
	"fmt"
	"time"
)

var ErrUnexpectedType = errors.New("circuit breaker returned unexpected type")

const (
	// DefaultConsecutiveFailures is the default threshold for consecutive failures (at least the one used in gobreaker)
	DefaultConsecutiveFailures = 5
)

type CircuitBreaker interface {
	Execute(fn func() error) error

	// ExecuteWithResult executes a function with a return value through the circuit breaker.
	// IMPORTANT: The calling code is responsible for type consistency when using type assertion.
	// Type mismatch will result in panic. Usage example:
	//   result, err := cb.ExecuteWithResult(func() (any, error) {
	//       return someFunc() // should return the expected type
	//   })
	//   typedResult := result.(ExpectedType) // calling code must be sure of the type
	ExecuteWithResult(fn func() (any, error)) (any, error)
}

// Config contains the configuration for Circuit Breaker
type Config interface {
	// Enabled determines whether the circuit breaker is enabled
	Enabled() bool

	Name() string

	// MaxRequests is the maximum number of requests allowed in half-open state
	// gobreaker default: 1 (if 0 is passed)
	MaxRequests() uint32

	// Interval is the cyclic period in closed state to reset internal counters
	// gobreaker default: 0 (counters are not reset if <= 0)
	Interval() time.Duration

	// Timeout is the period in open state after which the circuit breaker transitions to half-open
	// gobreaker default: 60 seconds (if <= 0 is passed)
	Timeout() time.Duration

	// ConsecutiveFailures is the threshold of consecutive failures to transition to open state
	// If 0 is passed, DefaultConsecutiveFailures will be used
	// If value N is specified, circuit breaker will open after N consecutive failures
	ConsecutiveFailures() uint32

	// BucketPeriod defines the time interval for each bucket in the rolling window strategy
	// If <= 0, the fixed window strategy is used
	BucketPeriod() time.Duration
}

// New
// ShouldIgnoreError is an optional function that determines if an error should be ignored
// by the circuit breaker.
// This is useful for filtering out expected errors (e.g. ErrNoRows in database operations).
func New(config Config, shouldIgnoreError func(err error) bool) CircuitBreaker {
	if !config.Enabled() {
		return &passthroughImpl{}
	}

	return newGobreakerImpl(config, shouldIgnoreError)
}

// Execute is a utility function that wraps circuit breaker Execute.
// If circuitBreaker is nil, it executes the function directly.
func Execute(circuitBreaker CircuitBreaker, fn func() error) error {
	if circuitBreaker != nil {
		return circuitBreaker.Execute(fn)
	}

	return fn()
}

// ExecuteWithResult is a generic utility function that wraps circuit breaker ExecuteWithResult
// with type safety and type assertion.
// It returns the fallbackValue on any error (circuit breaker error or type assertion error).
func ExecuteWithResult[T any](
	circuitBreaker CircuitBreaker,
	fn func() (T, error),
	fallbackValue T,
) (T, error) {
	if circuitBreaker != nil {
		result, err := circuitBreaker.ExecuteWithResult(func() (any, error) {
			return fn()
		})
		if err != nil {
			return fallbackValue, err
		}

		typedResult, ok := result.(T)
		if !ok {
			return fallbackValue, fmt.Errorf("%w: %T", ErrUnexpectedType, result)
		}

		return typedResult, nil
	}

	return fn()
}
