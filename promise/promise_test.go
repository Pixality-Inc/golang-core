package promise

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pixality-inc/golang-core/either"
	"github.com/stretchr/testify/require"
)

var (
	errPromiseTest    = errors.New("promise test error")
	errPromiseAnother = errors.New("another error")
)

func TestPromiseResolveGet(t *testing.T) {
	t.Parallel()

	prom := New[int]()

	require.NoError(t, prom.Resolve(42))
	require.True(t, prom.IsResolved())

	value, err := prom.Get(context.Background())

	require.NoError(t, err)
	require.Equal(t, 42, value)
}

func TestPromiseRejectGet(t *testing.T) {
	t.Parallel()

	prom := New[int]()

	require.NoError(t, prom.Reject(errPromiseTest))
	require.True(t, prom.IsResolved())

	value, err := prom.Get(context.Background())

	require.ErrorIs(t, err, errPromiseTest)
	require.Zero(t, value)
}

func TestPromiseGetWaitsUntilResolve(t *testing.T) {
	t.Parallel()

	prom := New[string]()
	resultCh := make(chan string, 1)
	errCh := make(chan error, 1)

	go func() {
		value, err := prom.Get(context.Background())
		resultCh <- value

		errCh <- err
	}()

	require.Never(t, func() bool {
		return len(resultCh) > 0 || len(errCh) > 0
	}, 20*time.Millisecond, time.Millisecond)

	require.NoError(t, prom.Resolve("ready"))
	require.Equal(t, "ready", receiveValue(t, resultCh))
	require.NoError(t, receiveValue(t, errCh))
}

func TestPromiseGetContextCanceledBeforeResolve(t *testing.T) {
	t.Parallel()

	prom := New[int]()
	impl := requirePromiseImpl(t, prom)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	value, err := prom.Get(ctx)

	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, value)
	require.False(t, prom.IsResolved())
	require.Empty(t, impl.channels)

	require.NoError(t, prom.Resolve(42))

	value, err = prom.Get(context.Background())
	require.NoError(t, err)
	require.Equal(t, 42, value)
}

func TestPromiseGetPrefersResolvedValueOverCanceledContext(t *testing.T) {
	t.Parallel()

	prom := New[int]()
	require.NoError(t, prom.Resolve(42))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	for range 100 {
		value, err := prom.Get(ctx)
		require.NoError(t, err)
		require.Equal(t, 42, value)
	}
}

func TestPromiseChanSubscribersBeforeResolve(t *testing.T) {
	t.Parallel()

	prom := New[string]()

	ch1 := prom.Chan()
	ch2 := prom.Chan()

	require.NoError(t, prom.Resolve("done"))

	requireEitherRight(t, receiveEither(t, ch1), "done")
	requireChannelClosed(t, ch1)
	requireEitherRight(t, receiveEither(t, ch2), "done")
	requireChannelClosed(t, ch2)
	require.True(t, prom.IsResolved())
}

func TestPromiseChanAfterResolve(t *testing.T) {
	t.Parallel()

	prom := New[string]()
	require.NoError(t, prom.Resolve("resolved"))

	ch := prom.Chan()

	requireEitherRight(t, receiveEither(t, ch), "resolved")
	requireChannelClosed(t, ch)
}

func TestPromiseChanAfterReject(t *testing.T) {
	t.Parallel()

	prom := New[string]()
	require.NoError(t, prom.Reject(errPromiseTest))

	ch := prom.Chan()

	requireEitherLeft(t, receiveEither(t, ch), errPromiseTest)
	requireChannelClosed(t, ch)
}

func TestPromiseResolveRejectAlreadyResolved(t *testing.T) {
	t.Parallel()

	prom := New[int]()

	require.NoError(t, prom.Resolve(10))
	require.ErrorIs(t, prom.Resolve(20), ErrAlreadyResolved)
	require.ErrorIs(t, prom.Reject(errPromiseTest), ErrAlreadyResolved)

	value, err := prom.Get(context.Background())
	require.NoError(t, err)
	require.Equal(t, 10, value)
}

func TestPromiseRejectResolveAlreadyResolved(t *testing.T) {
	t.Parallel()

	prom := New[int]()

	require.NoError(t, prom.Reject(errPromiseTest))
	require.ErrorIs(t, prom.Reject(errPromiseAnother), ErrAlreadyResolved)
	require.ErrorIs(t, prom.Resolve(20), ErrAlreadyResolved)

	value, err := prom.Get(context.Background())
	require.ErrorIs(t, err, errPromiseTest)
	require.Zero(t, value)
}

func TestPromiseRejectNilError(t *testing.T) {
	t.Parallel()

	prom := New[int]()

	require.ErrorIs(t, prom.Reject(nil), ErrNilError)
	require.False(t, prom.IsResolved())

	require.NoError(t, prom.Resolve(42))
}

func TestPromiseGetResolvedValueNotResolved(t *testing.T) {
	t.Parallel()

	prom := requirePromiseImpl(t, New[int]())

	value, err := prom.getResolvedValue(true).Value()

	require.ErrorIs(t, err, ErrNotResolved)
	require.Zero(t, value)
}

func TestPromiseGetResolvedValueMissingResult(t *testing.T) {
	t.Parallel()

	prom := requirePromiseImpl(t, New[int]())
	prom.resolved.Store(true)

	value, err := prom.getResolvedValue(true).Value()

	require.ErrorIs(t, err, ErrPromiseResolved)
	require.Zero(t, value)
}

func receiveEither[T any](t *testing.T, ch <-chan either.EitherError[T]) either.EitherError[T] {
	t.Helper()

	select {
	case value, ok := <-ch:
		require.True(t, ok)

		return value
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for promise channel")

		return nil
	}
}

func requirePromiseImpl[T any](t *testing.T, prom Promise[T]) *Impl[T] {
	t.Helper()

	impl, ok := prom.(*Impl[T])
	require.True(t, ok)

	return impl
}

func requireEitherRight[T comparable](t *testing.T, eith either.EitherError[T], expected T) {
	t.Helper()

	value, err := eith.Value()
	require.NoError(t, err)
	require.Equal(t, expected, value)
}

func requireEitherLeft[T any](t *testing.T, eith either.EitherError[T], expected error) {
	t.Helper()

	value, err := eith.Value()
	require.ErrorIs(t, err, expected)
	require.Zero(t, value)
}

func requireChannelClosed[T any](t *testing.T, ch <-chan either.EitherError[T]) {
	t.Helper()

	select {
	case _, ok := <-ch:
		require.False(t, ok)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for promise channel close")
	}
}

func receiveValue[T any](t *testing.T, ch <-chan T) T {
	t.Helper()

	select {
	case value := <-ch:
		return value
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for value")

		var defaultValue T

		return defaultValue
	}
}
