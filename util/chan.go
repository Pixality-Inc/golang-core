package util

import (
	"context"
	"errors"
	"time"
)

var (
	ErrContext       = errors.New("context")
	ErrChannelClosed = errors.New("channel closed")
)

func ReadFromChannelWithTimeout[T any](ctx context.Context, ch <-chan T, timeout time.Duration, defaultValue T) (T, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result, err := ReadFromChannel(ctx, ch, defaultValue)

	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return result, nil

	default:
		return result, err
	}
}

func ReadFromChannel[T any](ctx context.Context, ch <-chan T, defaultValue T) (T, error) {
	select {
	case <-ctx.Done():
		return defaultValue, errors.Join(ErrContext, ctx.Err())

	case v, ok := <-ch:
		if !ok {
			return defaultValue, ErrChannelClosed
		}

		return v, nil
	}
}
