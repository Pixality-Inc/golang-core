package errors_test

import (
	"fmt"
	"testing"

	"github.com/pixality-inc/golang-core/errors"
	"github.com/stretchr/testify/require"
)

var (
	errSentinel = errors.New("test.sentinel", "sentinel error")
	errCause    = errors.New("test.cause", "cause error")
)

func TestNew(t *testing.T) {
	t.Parallel()

	err := errors.New("some.code", "some message")

	require.Equal(t, errors.Code("some.code"), err.Code())
	require.Equal(t, "some message", err.Error())
	require.NotNil(t, err.Params())
	require.Empty(t, err.Params())
	require.NoError(t, err.Unwrap())
}

func TestNewWithCause(t *testing.T) {
	t.Parallel()

	err := errors.New("some.code", "some message", errors.WithCause(errCause))

	require.Equal(t, errCause, err.Unwrap())
	require.ErrorIs(t, err, errCause)
}

func TestNewWithParam(t *testing.T) {
	t.Parallel()

	err := errors.New("some.code", "some message", errors.WithParam("key", "value"))

	require.Equal(t, errors.Params{"key": "value"}, err.Params())
}

func TestNewWithParams(t *testing.T) {
	t.Parallel()

	params := errors.Params{"first": 1, "second": "two"}

	err := errors.New("some.code", "some message", errors.WithParams(params))

	require.Equal(t, params, err.Params())

	params["third"] = 3.0

	require.NotContains(t, err.Params(), "third")
}

func TestNewWithCombinedOptions(t *testing.T) {
	t.Parallel()

	err := errors.New(
		"some.code",
		"some message",
		errors.WithCause(errCause),
		errors.WithParams(errors.Params{"first": 1, "second": 2}),
		errors.WithParam("second", "overridden"),
		errors.WithParam("third", 3),
	)

	require.ErrorIs(t, err, errCause)
	require.Equal(t, errors.Params{"first": 1, "second": "overridden", "third": 3}, err.Params())
}

func TestThrow(t *testing.T) {
	t.Parallel()

	err := errors.New("some.code", "some message")

	require.Same(t, err, err.Throw())
}

func TestImplementsErrorInterface(t *testing.T) {
	t.Parallel()

	var err errors.Error = errors.New("some.code", "some message")

	require.Equal(t, errors.Code("some.code"), err.Code())
}

func TestIsThroughJoin(t *testing.T) {
	t.Parallel()

	wrapped := errors.Join(errSentinel, errCause)

	require.ErrorIs(t, wrapped, errSentinel)
	require.ErrorIs(t, wrapped, errCause)
}

func TestIsThroughCauseChain(t *testing.T) {
	t.Parallel()

	inner := errors.New("test.inner", "inner", errors.WithCause(errCause))
	outer := errors.New("test.outer", "outer", errors.WithCause(inner))

	require.ErrorIs(t, outer, inner)
	require.ErrorIs(t, outer, errCause)
	require.NotErrorIs(t, outer, errSentinel)
}

func TestAs(t *testing.T) {
	t.Parallel()

	wrapped := fmt.Errorf("wrapped: %w", errSentinel)

	var implErr *errors.ImplError
	require.ErrorAs(t, wrapped, &implErr)
	require.Equal(t, errors.Code("test.sentinel"), implErr.Code())
}

func TestUnwrapAlias(t *testing.T) {
	t.Parallel()

	err := errors.New("some.code", "some message", errors.WithCause(errCause))

	require.Equal(t, errCause, errors.Unwrap(err))
	require.NoError(t, errors.Unwrap(errSentinel))
}
