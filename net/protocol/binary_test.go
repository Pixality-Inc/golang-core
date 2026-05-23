package protocol

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBinaryMarshalReturnsContextErrorWhenCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := NewBinary().Marshal(ctx, []byte("message"))
	require.ErrorIs(t, err, context.Canceled)
}

func TestBinaryReadClosesImmediatelyWhenContextCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	messages, err := NewBinary().Read(ctx, strings.NewReader("message"))
	require.NoError(t, err)
	requireClosed(t, messages)
}

func TestBinaryReadStopsWhenContextCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	reader, writer := io.Pipe()
	defer func() {
		_ = writer.Close()
	}()

	messages, err := NewBinary().Read(ctx, reader)
	require.NoError(t, err)

	cancel()
	requireClosed(t, messages)
}

func requireClosed[T any](t *testing.T, ch <-chan T) {
	t.Helper()

	select {
	case _, ok := <-ch:
		require.False(t, ok)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for channel close")
	}
}
