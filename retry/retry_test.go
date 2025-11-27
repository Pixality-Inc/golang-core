package retry

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pixality-inc/golang-core/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testSuccess         = "success"
	testGenericError    = "generic error"
	testTemporaryError  = "temporary error"
	testPersistentError = "persistent error"
	testRetryError      = "error"
)

var (
	errGeneric    = errors.New(testGenericError)
	errTemporary  = errors.New(testTemporaryError)
	errPersistent = errors.New(testPersistentError)
	errRetry      = errors.New(testRetryError)
)

func TestShouldRetry_5xxStatusCodes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusCode int
		expected   bool
	}{
		{"500 internal server error", 500, true},
		{"502 bad gateway", 502, true},
		{"503 service unavailable", 503, true},
		{"504 gateway timeout", 504, true},
		{"599 custom 5xx", 599, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := ShouldRetry(tc.statusCode, nil)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestShouldRetry_429StatusCode(t *testing.T) {
	t.Parallel()

	result := ShouldRetry(429, nil)
	assert.True(t, result)
}

func TestShouldRetry_4xxStatusCodes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusCode int
		expected   bool
	}{
		{"400 bad request", 400, false},
		{"401 unauthorized", 401, false},
		{"403 forbidden", 403, false},
		{"404 not found", 404, false},
		{"422 unprocessable entity", 422, false},
		{"499 custom 4xx", 499, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := ShouldRetry(tc.statusCode, nil)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestShouldRetry_2xxStatusCodes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusCode int
		expected   bool
	}{
		{"200 ok", 200, false},
		{"201 created", 201, false},
		{"204 no content", 204, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := ShouldRetry(tc.statusCode, nil)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestShouldRetry_NetworkErrors(t *testing.T) {
	t.Parallel()

	t.Run("network timeout error", func(t *testing.T) {
		t.Parallel()

		err := &net.DNSError{IsTimeout: true}
		result := ShouldRetry(0, err)
		assert.True(t, result)
	})

	t.Run("network operation error", func(t *testing.T) {
		t.Parallel()

		err := &net.OpError{Op: "dial"}
		result := ShouldRetry(0, err)
		assert.True(t, result)
	})
}

func TestShouldRetry_ContextErrors(t *testing.T) {
	t.Parallel()

	t.Run("context deadline exceeded", func(t *testing.T) {
		t.Parallel()

		result := ShouldRetry(0, context.DeadlineExceeded)
		assert.False(t, result)
	})

	t.Run("context canceled", func(t *testing.T) {
		t.Parallel()

		result := ShouldRetry(0, context.Canceled)
		assert.False(t, result)
	})
}

func TestShouldRetry_OtherErrors(t *testing.T) {
	t.Parallel()

	t.Run("generic error", func(t *testing.T) {
		t.Parallel()

		result := ShouldRetry(0, errGeneric)
		assert.True(t, result)
	})
}

func TestCalculateBackoff_ExponentialBackoff(t *testing.T) {
	t.Parallel()

	policy := &PolicyImpl{
		InitialIntervalValue:    100 * time.Millisecond,
		BackoffCoefficientValue: 2.0,
		MaxIntervalValue:        10 * time.Second,
	}

	testCases := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 100 * time.Millisecond},
		{1, 200 * time.Millisecond},
		{2, 400 * time.Millisecond},
		{3, 800 * time.Millisecond},
		{4, 1600 * time.Millisecond},
	}

	for _, tc := range testCases {
		t.Run("attempt "+string(rune(tc.attempt+'0')), func(t *testing.T) {
			t.Parallel()

			result := CalculateBackoff(tc.attempt, policy)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCalculateBackoff_MaxInterval(t *testing.T) {
	t.Parallel()

	policy := &PolicyImpl{
		InitialIntervalValue:    1 * time.Second,
		BackoffCoefficientValue: 10.0,
		MaxIntervalValue:        5 * time.Second,
	}

	result := CalculateBackoff(5, policy)
	assert.Equal(t, 5*time.Second, result)
}

func TestCalculateBackoff_NilPolicy(t *testing.T) {
	t.Parallel()

	result := CalculateBackoff(3, nil)
	assert.Equal(t, time.Duration(0), result)
}

func TestCalculateBackoff_DifferentCoefficients(t *testing.T) {
	t.Parallel()

	t.Run("coefficient 1.5", func(t *testing.T) {
		t.Parallel()

		policy := &PolicyImpl{
			InitialIntervalValue:    100 * time.Millisecond,
			BackoffCoefficientValue: 1.5,
			MaxIntervalValue:        10 * time.Second,
		}

		result := CalculateBackoff(2, policy)
		assert.Equal(t, 225*time.Millisecond, result)
	})

	t.Run("coefficient 3.0", func(t *testing.T) {
		t.Parallel()

		policy := &PolicyImpl{
			InitialIntervalValue:    100 * time.Millisecond,
			BackoffCoefficientValue: 3.0,
			MaxIntervalValue:        10 * time.Second,
		}

		result := CalculateBackoff(2, policy)
		assert.Equal(t, 900*time.Millisecond, result)
	})
}

func TestDo_SuccessFirstAttempt(t *testing.T) {
	t.Parallel()

	policy := &PolicyImpl{
		EnabledValue:            true,
		MaxAttemptsValue:        3,
		InitialIntervalValue:    10 * time.Millisecond,
		BackoffCoefficientValue: 2.0,
		MaxIntervalValue:        100 * time.Millisecond,
	}

	log := logger.NewLoggableImplWithService("test")
	ctx := context.Background()

	var attempts atomic.Int32

	result, err := Do(ctx, policy, log, func() (string, error) {
		attempts.Add(1)

		return testSuccess, nil
	})

	require.NoError(t, err)
	assert.Equal(t, testSuccess, result)
	assert.Equal(t, int32(1), attempts.Load())
}

func TestDo_RetryWithSuccess(t *testing.T) {
	t.Parallel()

	policy := &PolicyImpl{
		EnabledValue:            true,
		MaxAttemptsValue:        3,
		InitialIntervalValue:    10 * time.Millisecond,
		BackoffCoefficientValue: 2.0,
		MaxIntervalValue:        100 * time.Millisecond,
	}

	log := logger.NewLoggableImplWithService("test")
	ctx := context.Background()

	var attempts atomic.Int32

	result, err := Do(ctx, policy, log, func() (string, error) {
		count := attempts.Add(1)
		if count < 3 {
			return "", errTemporary
		}

		return testSuccess, nil
	})

	require.NoError(t, err)
	assert.Equal(t, testSuccess, result)
	assert.Equal(t, int32(3), attempts.Load())
}

func TestDo_ExhaustAllAttempts(t *testing.T) {
	t.Parallel()

	policy := &PolicyImpl{
		EnabledValue:            true,
		MaxAttemptsValue:        3,
		InitialIntervalValue:    10 * time.Millisecond,
		BackoffCoefficientValue: 2.0,
		MaxIntervalValue:        100 * time.Millisecond,
	}

	log := logger.NewLoggableImplWithService("test")
	ctx := context.Background()

	var attempts atomic.Int32

	result, err := Do(ctx, policy, log, func() (string, error) {
		attempts.Add(1)

		return "", errPersistent
	})

	require.ErrorIs(t, err, errPersistent)
	assert.Empty(t, result)
	assert.Equal(t, int32(3), attempts.Load())
}

func TestDo_ContextCancellation(t *testing.T) {
	t.Parallel()

	policy := &PolicyImpl{
		EnabledValue:            true,
		MaxAttemptsValue:        10,
		InitialIntervalValue:    50 * time.Millisecond,
		BackoffCoefficientValue: 2.0,
		MaxIntervalValue:        200 * time.Millisecond,
	}

	log := logger.NewLoggableImplWithService("test")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	var attempts atomic.Int32

	_, err := Do(ctx, policy, log, func() (string, error) {
		attempts.Add(1)

		return "", errRetry
	})

	require.Error(t, err)
	assert.Less(t, attempts.Load(), int32(10))
}

func TestDo_NilPolicy(t *testing.T) {
	t.Parallel()

	log := logger.NewLoggableImplWithService("test")
	ctx := context.Background()

	var attempts atomic.Int32

	result, err := Do(ctx, nil, log, func() (string, error) {
		attempts.Add(1)

		return testSuccess, nil
	})

	require.NoError(t, err)
	assert.Equal(t, testSuccess, result)
	assert.Equal(t, int32(1), attempts.Load())
}

func TestDo_MaxAttemptsOne(t *testing.T) {
	t.Parallel()

	policy := &PolicyImpl{
		MaxAttemptsValue:        1,
		InitialIntervalValue:    10 * time.Millisecond,
		BackoffCoefficientValue: 2.0,
		MaxIntervalValue:        100 * time.Millisecond,
	}

	log := logger.NewLoggableImplWithService("test")
	ctx := context.Background()

	var attempts atomic.Int32

	result, err := Do(ctx, policy, log, func() (string, error) {
		attempts.Add(1)

		return testSuccess, nil
	})

	require.NoError(t, err)
	assert.Equal(t, testSuccess, result)
	assert.Equal(t, int32(1), attempts.Load())
}

func TestDoWithCondition_CustomShouldRetry(t *testing.T) {
	t.Parallel()

	policy := &PolicyImpl{
		EnabledValue:            true,
		MaxAttemptsValue:        3,
		InitialIntervalValue:    10 * time.Millisecond,
		BackoffCoefficientValue: 2.0,
		MaxIntervalValue:        100 * time.Millisecond,
	}

	log := logger.NewLoggableImplWithService("test")
	ctx := context.Background()

	var attempts atomic.Int32

	type response struct {
		value int
	}

	result, err := DoWithCondition(
		ctx,
		policy,
		log,
		func() (response, error) {
			count := attempts.Add(1)

			return response{value: int(count)}, nil
		},
		func(r response, err error) bool {
			return r.value < 3
		},
	)

	require.NoError(t, err)
	assert.Equal(t, 3, result.value)
	assert.Equal(t, int32(3), attempts.Load())
}

func TestDoWithCondition_SuccessFirstAttempt(t *testing.T) {
	t.Parallel()

	policy := &PolicyImpl{
		EnabledValue:            true,
		MaxAttemptsValue:        3,
		InitialIntervalValue:    10 * time.Millisecond,
		BackoffCoefficientValue: 2.0,
		MaxIntervalValue:        100 * time.Millisecond,
	}

	log := logger.NewLoggableImplWithService("test")
	ctx := context.Background()

	var attempts atomic.Int32

	result, err := DoWithCondition(
		ctx,
		policy,
		log,
		func() (string, error) {
			attempts.Add(1)

			return testSuccess, nil
		},
		func(r string, err error) bool {
			return err != nil
		},
	)

	require.NoError(t, err)
	assert.Equal(t, testSuccess, result)
	assert.Equal(t, int32(1), attempts.Load())
}

func TestDoWithCondition_RetryWithSuccess(t *testing.T) {
	t.Parallel()

	policy := &PolicyImpl{
		EnabledValue:            true,
		MaxAttemptsValue:        3,
		InitialIntervalValue:    10 * time.Millisecond,
		BackoffCoefficientValue: 2.0,
		MaxIntervalValue:        100 * time.Millisecond,
	}

	log := logger.NewLoggableImplWithService("test")
	ctx := context.Background()

	var attempts atomic.Int32

	result, err := DoWithCondition(
		ctx,
		policy,
		log,
		func() (string, error) {
			count := attempts.Add(1)
			if count < 2 {
				return "", errTemporary
			}

			return testSuccess, nil
		},
		func(r string, err error) bool {
			return err != nil
		},
	)

	require.NoError(t, err)
	assert.Equal(t, testSuccess, result)
	assert.Equal(t, int32(2), attempts.Load())
}

func TestDoWithCondition_ContextCancellation(t *testing.T) {
	t.Parallel()

	policy := &PolicyImpl{
		EnabledValue:            true,
		MaxAttemptsValue:        10,
		InitialIntervalValue:    50 * time.Millisecond,
		BackoffCoefficientValue: 2.0,
		MaxIntervalValue:        200 * time.Millisecond,
	}

	log := logger.NewLoggableImplWithService("test")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	var attempts atomic.Int32

	_, err := DoWithCondition(
		ctx,
		policy,
		log,
		func() (string, error) {
			attempts.Add(1)

			return "", errRetry
		},
		func(r string, err error) bool {
			return true
		},
	)

	require.Error(t, err)
	assert.Less(t, attempts.Load(), int32(10))
}

func TestDoWithCondition_NilPolicy(t *testing.T) {
	t.Parallel()

	log := logger.NewLoggableImplWithService("test")
	ctx := context.Background()

	var attempts atomic.Int32

	result, err := DoWithCondition(
		ctx,
		nil,
		log,
		func() (string, error) {
			attempts.Add(1)

			return testSuccess, nil
		},
		func(r string, err error) bool {
			return false
		},
	)

	require.NoError(t, err)
	assert.Equal(t, testSuccess, result)
	assert.Equal(t, int32(1), attempts.Load())
}

func TestNewPolicy_DefaultValues(t *testing.T) {
	t.Parallel()

	policy := NewPolicy()

	assert.Equal(t, 3, policy.MaxAttempts())
	assert.Equal(t, 100*time.Millisecond, policy.InitialInterval())
	assert.InEpsilon(t, 2.0, policy.BackoffCoefficient(), 0.001)
	assert.Equal(t, 5*time.Second, policy.MaxInterval())
}

func TestNewPolicy_WithCustomOptions(t *testing.T) {
	t.Parallel()

	policy := NewPolicy(
		WithMaxAttempts(5),
		WithInitialInterval(200*time.Millisecond),
		WithBackoffCoefficient(1.5),
		WithMaxInterval(10*time.Second),
	)

	assert.Equal(t, 5, policy.MaxAttempts())
	assert.Equal(t, 200*time.Millisecond, policy.InitialInterval())
	assert.InEpsilon(t, 1.5, policy.BackoffCoefficient(), 0.001)
	assert.Equal(t, 10*time.Second, policy.MaxInterval())
}

func TestNewPolicy_PartialOptions(t *testing.T) {
	t.Parallel()

	policy := NewPolicy(
		WithMaxAttempts(10),
		WithInitialInterval(500*time.Millisecond),
	)

	assert.Equal(t, 10, policy.MaxAttempts())
	assert.Equal(t, 500*time.Millisecond, policy.InitialInterval())
	assert.InEpsilon(t, 2.0, policy.BackoffCoefficient(), 0.001)
	assert.Equal(t, 5*time.Second, policy.MaxInterval())
}

func TestConfig_AsPolicy(t *testing.T) {
	t.Parallel()

	config := &ConfigYaml{
		MaxAttemptsValue:        7,
		InitialIntervalValue:    250 * time.Millisecond,
		BackoffCoefficientValue: 3.0,
		MaxIntervalValue:        15 * time.Second,
	}

	assert.Equal(t, 7, config.MaxAttempts())
	assert.Equal(t, 250*time.Millisecond, config.InitialInterval())
	assert.InEpsilon(t, 3.0, config.BackoffCoefficient(), 0.001)
	assert.Equal(t, 15*time.Second, config.MaxInterval())
}

func TestPolicyImpl_AsPolicy(t *testing.T) {
	t.Parallel()

	impl := &PolicyImpl{
		MaxAttemptsValue:        4,
		InitialIntervalValue:    150 * time.Millisecond,
		BackoffCoefficientValue: 2.5,
		MaxIntervalValue:        8 * time.Second,
	}

	assert.Equal(t, 4, impl.MaxAttempts())
	assert.Equal(t, 150*time.Millisecond, impl.InitialInterval())
	assert.InEpsilon(t, 2.5, impl.BackoffCoefficient(), 0.001)
	assert.Equal(t, 8*time.Second, impl.MaxInterval())
}
