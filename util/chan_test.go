package util

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_ReadFromChannel(t *testing.T) {
	t.Parallel()

	ch := make(chan int, 1)
	ch <- 1

	result, err := ReadFromChannel(context.Background(), ch, 0)
	require.NoError(t, err)
	require.Equal(t, 1, result)
}

func Test_ReadFromChannel_Timeout(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)

	ch := make(chan int, 1)

	t.Cleanup(func() {
		cancel()
	})

	result, err := ReadFromChannel(ctx, ch, 0)
	require.ErrorIs(t, err, ErrContext)
	require.Equal(t, 0, result)
}

func Test_ReadFromChannel_Closed(t *testing.T) {
	t.Parallel()

	ch := make(chan int, 1)
	close(ch)

	result, err := ReadFromChannel(context.Background(), ch, 0)
	require.ErrorIs(t, err, ErrChannelClosed)
	require.Equal(t, 0, result)
}

func Test_ReadFromChannelWithTimeout(t *testing.T) {
	t.Parallel()

	ch := make(chan int, 1)
	ch <- 1

	result, err := ReadFromChannelWithTimeout(context.Background(), ch, 100*time.Millisecond, 0)
	require.NoError(t, err)
	require.Equal(t, 1, result)
}

func Test_ReadFromChannelWithTimeout_Cancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	ch := make(chan int, 1)

	cancel()

	result, err := ReadFromChannelWithTimeout(ctx, ch, 100*time.Millisecond, 0)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 0, result)
}

func Test_ReadFromChannelWithTimeout_Timeout(t *testing.T) {
	t.Parallel()

	ch := make(chan int, 1)

	result, err := ReadFromChannelWithTimeout(context.Background(), ch, 100*time.Millisecond, 0)
	require.NoError(t, err)
	require.Equal(t, 0, result)
}

func Test_ReadFromChannelWithTimeout_Closed(t *testing.T) {
	t.Parallel()

	ch := make(chan int, 1)
	close(ch)

	result, err := ReadFromChannelWithTimeout(context.Background(), ch, 100*time.Millisecond, 0)
	require.ErrorIs(t, err, ErrChannelClosed)
	require.Equal(t, 0, result)
}
