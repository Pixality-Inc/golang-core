package future

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pixality-inc/golang-core/either"
	"github.com/stretchr/testify/require"
)

var (
	errFutureTest                = errors.New("future test error")
	errFutureContextValueMissing = errors.New("future context value missing")
)

func TestFutureGetSuccess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	fut := New(ctx, func(ctx context.Context) (int, error) {
		return 42, nil
	})

	value, err := fut.Get(ctx)

	require.NoError(t, err)
	require.Equal(t, 42, value)
	require.Eventually(t, fut.IsResolved, time.Second, time.Millisecond)
}

func TestFutureGetError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	fut := New(ctx, func(ctx context.Context) (int, error) {
		return 0, errFutureTest
	})

	value, err := fut.Get(ctx)

	require.ErrorIs(t, err, errFutureTest)
	require.Zero(t, value)
	require.Eventually(t, fut.IsResolved, time.Second, time.Millisecond)
}

func TestFutureGetContextCanceled(t *testing.T) {
	t.Parallel()

	started := make(chan struct{})
	release := make(chan struct{})

	fut := New(t.Context(), func(ctx context.Context) (int, error) {
		close(started)

		select {
		case <-release:
			return 42, nil
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	})

	waitForClosed(t, started)

	getCtx, cancelGet := context.WithCancel(context.Background())
	cancelGet()

	value, err := fut.Get(getCtx)

	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, value)
	require.False(t, fut.IsResolved())

	close(release)
	require.Eventually(t, fut.IsResolved, time.Second, time.Millisecond)
}

func TestFuturePassesRunContextToBody(t *testing.T) {
	t.Parallel()

	type contextKey string

	ctx := context.WithValue(context.Background(), contextKey("name"), "future-context")

	fut := New(ctx, func(ctx context.Context) (string, error) {
		value, ok := ctx.Value(contextKey("name")).(string)
		if !ok {
			return "", errFutureContextValueMissing
		}

		return value, nil
	})

	value, err := fut.Get(context.Background())

	require.NoError(t, err)
	require.Equal(t, "future-context", value)
}

func TestFutureChanReceivesBodyResult(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	release := make(chan struct{})

	fut := New(ctx, func(ctx context.Context) (string, error) {
		<-release

		return "done", nil
	})

	ch := fut.Chan()

	close(release)

	requireEitherRight(t, receiveEither(t, ch), "done")
	requireChannelClosed(t, ch)
}

func TestFutureChanReceivesBodyError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	fut := New(ctx, func(ctx context.Context) (string, error) {
		return "", errFutureTest
	})

	ch := fut.Chan()

	requireEitherLeft(t, receiveEither(t, ch), errFutureTest)
	requireChannelClosed(t, ch)
}

func receiveEither[T any](t *testing.T, ch <-chan either.EitherError[T]) either.EitherError[T] {
	t.Helper()

	select {
	case value, ok := <-ch:
		require.True(t, ok)

		return value
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for future channel")

		return nil
	}
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
		t.Fatal("timed out waiting for future channel close")
	}
}

func waitForClosed(t *testing.T, ch <-chan struct{}) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for future body start")
	}
}
