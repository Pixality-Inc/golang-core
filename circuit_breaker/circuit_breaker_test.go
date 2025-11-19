package circuit_breaker

import (
	"errors"
	"testing"
	"time"

	"github.com/sony/gobreaker/v2"
	"github.com/stretchr/testify/require"
)

var errTest = errors.New("test error")

func TestCircuitBreaker_Disabled(t *testing.T) {
	t.Parallel()

	cb := New(&ConfigYaml{
		EnabledValue:             false,
		NameValue:                "test-disabled",
		ConsecutiveFailuresValue: 2,
	}, nil)

	// Even with many errors, the circuit breaker should not trip
	for range 10 {
		err := cb.Execute(func() error {
			return errTest
		})
		require.Error(t, err)
		require.Equal(t, errTest, err, "should get original error")
	}
}

func TestCircuitBreaker_OpensAfterFailures(t *testing.T) {
	t.Parallel()

	cb := New(&ConfigYaml{
		EnabledValue:             true,
		NameValue:                "test-opens",
		MaxRequestsValue:         1,
		IntervalValue:            10 * time.Second,
		TimeoutValue:             100 * time.Millisecond,
		ConsecutiveFailuresValue: 3,
		BucketPeriodValue:        1 * time.Second,
	}, nil)

	// First 3 calls should pass through the function and return test error
	for i := range 3 {
		err := cb.Execute(func() error {
			return errTest
		})
		require.Error(t, err)
		require.Equal(t, errTest, err, "call %d should get original error", i+1)
	}

	// 4th call should get error from circuit breaker (open)
	err := cb.Execute(func() error {
		return errTest
	})
	require.Error(t, err)
	require.Equal(t, gobreaker.ErrOpenState, err, "circuit breaker should be open")
}

func TestCircuitBreaker_HalfOpenToClosedTransition(t *testing.T) {
	t.Parallel()

	cb := New(&ConfigYaml{
		EnabledValue:             true,
		NameValue:                "test-half-open",
		MaxRequestsValue:         1,
		IntervalValue:            10 * time.Second,
		TimeoutValue:             100 * time.Millisecond, // short timeout for fast test
		ConsecutiveFailuresValue: 2,
		BucketPeriodValue:        1 * time.Second,
	}, nil)

	// Open circuit breaker
	for range 2 {
		err := cb.Execute(func() error {
			return errTest
		})
		require.Error(t, err)
	}

	// Verify that circuit breaker is open
	err := cb.Execute(func() error {
		return errTest
	})
	require.Equal(t, gobreaker.ErrOpenState, err, "circuit breaker should be open")

	// Wait for transition to half-open
	time.Sleep(150 * time.Millisecond)

	// Successful call should close circuit breaker
	err = cb.Execute(func() error {
		return nil // success
	})
	require.NoError(t, err, "successful call in half-open state should close circuit breaker")

	// Next call should pass (circuit breaker closed)
	err = cb.Execute(func() error {
		return nil
	})
	require.NoError(t, err, "circuit breaker should be closed")
}

func TestCircuitBreaker_Success(t *testing.T) {
	t.Parallel()

	cb := New(&ConfigYaml{
		EnabledValue:             true,
		NameValue:                "test-success",
		MaxRequestsValue:         5,
		IntervalValue:            10 * time.Second,
		TimeoutValue:             1 * time.Second,
		ConsecutiveFailuresValue: 5,
		BucketPeriodValue:        1 * time.Second,
	}, nil)

	for i := range 10 {
		err := cb.Execute(func() error {
			return nil
		})
		require.NoError(t, err, "call %d should succeed", i+1)
	}
}

func TestCircuitBreaker_MixedResults(t *testing.T) {
	t.Parallel()

	cb := New(&ConfigYaml{
		EnabledValue:             true,
		NameValue:                "test-mixed",
		MaxRequestsValue:         5,
		IntervalValue:            10 * time.Second,
		TimeoutValue:             100 * time.Millisecond,
		ConsecutiveFailuresValue: 3,
		BucketPeriodValue:        1 * time.Second,
	}, nil)

	// 2 errors, 1 success - circuit breaker should not open
	err := cb.Execute(func() error {
		return errTest
	})
	require.Error(t, err)

	err = cb.Execute(func() error {
		return errTest
	})
	require.Error(t, err)

	err = cb.Execute(func() error {
		return nil // success resets consecutive error counter
	})
	require.NoError(t, err)

	// 2 more errors - circuit breaker should still not open
	err = cb.Execute(func() error {
		return errTest
	})
	require.Error(t, err)
	require.Equal(t, errTest, err, "should get original error")

	err = cb.Execute(func() error {
		return errTest
	})
	require.Error(t, err)
	require.Equal(t, errTest, err, "should get original error")

	// 3rd consecutive error - circuit breaker will open
	err = cb.Execute(func() error {
		return errTest
	})
	require.Error(t, err)
	require.Equal(t, errTest, err, "should get original error before opening")

	// Next call will get error from circuit breaker
	err = cb.Execute(func() error {
		return errTest
	})
	require.Equal(t, gobreaker.ErrOpenState, err, "circuit breaker should be open")
}

func TestCircuitBreaker_DefaultConsecutiveFailures(t *testing.T) {
	t.Parallel()

	cb := New(&ConfigYaml{
		EnabledValue:             true,
		NameValue:                "test-default-failures",
		MaxRequestsValue:         1,
		IntervalValue:            10 * time.Second,
		TimeoutValue:             100 * time.Millisecond,
		ConsecutiveFailuresValue: 0, // Should use DefaultConsecutiveFailures (5)
		BucketPeriodValue:        1 * time.Second,
	}, nil)

	for i := range DefaultConsecutiveFailures {
		err := cb.Execute(func() error {
			return errTest
		})
		require.Error(t, err)
		require.Equal(t, errTest, err, "call %d should get original error", i+1)
	}

	err := cb.Execute(func() error {
		return errTest
	})
	require.Error(t, err)
	require.Equal(t, gobreaker.ErrOpenState, err, "circuit breaker should be open")
}

func TestExecute_WithCircuitBreaker(t *testing.T) {
	t.Parallel()

	cb := New(&ConfigYaml{
		EnabledValue:             true,
		NameValue:                "test-execute",
		ConsecutiveFailuresValue: 2,
	}, nil)

	// Success case
	err := Execute(cb, func() error {
		return nil
	})
	require.NoError(t, err)

	// Error case
	err = Execute(cb, func() error {
		return errTest
	})
	require.Error(t, err)
	require.Equal(t, errTest, err)
}

func TestExecute_WithNilCircuitBreaker(t *testing.T) {
	t.Parallel()

	// Success case
	err := Execute(nil, func() error {
		return nil
	})
	require.NoError(t, err)

	// Error case
	err = Execute(nil, func() error {
		return errTest
	})
	require.Error(t, err)
	require.Equal(t, errTest, err)
}

func TestExecuteWithResult_WithCircuitBreaker(t *testing.T) {
	t.Parallel()

	cb := New(&ConfigYaml{
		EnabledValue:             true,
		NameValue:                "test-execute-with-result",
		ConsecutiveFailuresValue: 2,
	}, nil)

	// Success case
	result, err := ExecuteWithResult(cb, func() (string, error) {
		return "success", nil
	}, "fallback")
	require.NoError(t, err)
	require.Equal(t, "success", result)

	// Error case
	result, err = ExecuteWithResult(cb, func() (string, error) {
		return "", errTest
	}, "fallback")
	require.Error(t, err)
	require.Equal(t, errTest, err)
	require.Equal(t, "fallback", result)
}

func TestExecuteWithResult_WithNilCircuitBreaker(t *testing.T) {
	t.Parallel()

	// Success case
	result, err := ExecuteWithResult(nil, func() (int, error) {
		return 42, nil
	}, 0)
	require.NoError(t, err)
	require.Equal(t, 42, result)

	// Error case
	result, err = ExecuteWithResult(nil, func() (int, error) {
		return 0, errTest
	}, 99)
	require.Error(t, err)
	require.Equal(t, errTest, err)
	require.Equal(t, 0, result)
}

type fakeCircuitBreakerWrongType struct{}

func (f *fakeCircuitBreakerWrongType) Execute(fn func() error) error {
	return fn()
}

func (f *fakeCircuitBreakerWrongType) ExecuteWithResult(fn func() (any, error)) (any, error) {
	// Return wrong type to trigger type assertion error
	return "wrong-type", nil
}

func TestExecuteWithResult_TypeAssertionError(t *testing.T) {
	t.Parallel()

	cb := &fakeCircuitBreakerWrongType{}

	result, err := ExecuteWithResult(cb, func() (int, error) {
		return 42, nil
	}, 0)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrUnexpectedType)
	require.Equal(t, 0, result)
}
