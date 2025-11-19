package redis

import (
	"context"
	"errors"
	"testing"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

var (
	errWrongType         = errors.New("WRONGTYPE Operation against a key holding the wrong kind of value")
	errNoAuth            = errors.New("NOAUTH Authentication required")
	errWrongPass         = errors.New("ERR invalid password (WRONGPASS)")
	errNoPerm            = errors.New("NOPERM this user has no permissions")
	errReadOnly          = errors.New("READONLY You can't write against a read only replica")
	errMasterDown        = errors.New("MASTERDOWN Link with MASTER is down and replica-serve-stale-data is set to 'no'")
	errClusterDown       = errors.New("CLUSTERDOWN Hash slot not served")
	errLoading           = errors.New("LOADING Redis is loading the dataset in memory")
	errMaxClients        = errors.New("ERR max number of clients reached")
	errSomeGeneric       = errors.New("some generic error")
	errConnectionTimeout = errors.New("i/o timeout")
)

func TestShouldIgnoreErrorForCircuitBreaker_RedisNil(t *testing.T) {
	t.Parallel()

	shouldIgnore := ShouldIgnoreErrorForCircuitBreaker(goredis.Nil)
	require.True(t, shouldIgnore, "redis.Nil should be ignored")
}

func TestShouldIgnoreErrorForCircuitBreaker_ContextCanceled(t *testing.T) {
	t.Parallel()

	shouldIgnore := ShouldIgnoreErrorForCircuitBreaker(context.Canceled)
	require.True(t, shouldIgnore, "context.Canceled should be ignored")
}

func TestShouldIgnoreErrorForCircuitBreaker_WrongType(t *testing.T) {
	t.Parallel()

	shouldIgnore := ShouldIgnoreErrorForCircuitBreaker(errWrongType)
	require.True(t, shouldIgnore, "WRONGTYPE should be ignored (application error)")
}

func TestShouldIgnoreErrorForCircuitBreaker_AuthErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		err  error
		desc string
	}{
		{
			"noauth",
			errNoAuth,
			"authentication required",
		},
		{
			"wrongpass",
			errWrongPass,
			"wrong password",
		},
		{
			"noperm",
			errNoPerm,
			"no permissions",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			shouldIgnore := ShouldIgnoreErrorForCircuitBreaker(testCase.err)
			require.True(t, shouldIgnore, "%s should be ignored (config error)", testCase.desc)
		})
	}
}

func TestShouldIgnoreErrorForCircuitBreaker_ConnectionErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		err  error
		desc string
	}{
		{
			"closed",
			goredis.ErrClosed,
			"client closed",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			shouldIgnore := ShouldIgnoreErrorForCircuitBreaker(testCase.err)
			require.False(t, shouldIgnore, "%s SHOULD trigger circuit breaker", testCase.desc)
		})
	}
}

func TestShouldIgnoreErrorForCircuitBreaker_ClusterErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		err  error
		desc string
	}{
		{
			"readonly",
			errReadOnly,
			"readonly replica",
		},
		{
			"masterdown",
			errMasterDown,
			"master down",
		},
		{
			"clusterdown",
			errClusterDown,
			"cluster down",
		},
		{
			"loading",
			errLoading,
			"loading dataset",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			shouldIgnore := ShouldIgnoreErrorForCircuitBreaker(testCase.err)
			require.False(t, shouldIgnore, "%s SHOULD trigger circuit breaker", testCase.desc)
		})
	}
}

func TestShouldIgnoreErrorForCircuitBreaker_MaxClients(t *testing.T) {
	t.Parallel()

	shouldIgnore := ShouldIgnoreErrorForCircuitBreaker(errMaxClients)
	require.False(t, shouldIgnore, "max clients error SHOULD trigger circuit breaker")
}

func TestShouldIgnoreErrorForCircuitBreaker_GenericErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		err  error
		desc string
	}{
		{
			"generic_error",
			errSomeGeneric,
			"generic error",
		},
		{
			"connection_timeout",
			errConnectionTimeout,
			"connection timeout",
		},
		{
			"context_deadline",
			context.DeadlineExceeded,
			"context deadline exceeded",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			shouldIgnore := ShouldIgnoreErrorForCircuitBreaker(testCase.err)
			require.False(t, shouldIgnore, "%s SHOULD trigger circuit breaker", testCase.desc)
		})
	}
}

func TestShouldIgnoreErrorForCircuitBreaker_NilError(t *testing.T) {
	t.Parallel()

	shouldIgnore := ShouldIgnoreErrorForCircuitBreaker(nil)
	require.False(t, shouldIgnore, "nil error should return false")
}
