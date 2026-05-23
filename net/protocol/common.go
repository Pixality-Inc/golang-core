package protocol

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"
)

const DefaultBufferSize = 32 * 1024

var errInvalidProtocolBufferSize = errors.New("invalid protocol buffer size")

type readDeadliner interface {
	SetReadDeadline(deadline time.Time) error
}

func readChunks(ctx context.Context, reader io.Reader, bufferSize int) (chan []byte, error) {
	if bufferSize <= 0 {
		return nil, fmt.Errorf("%w: %d", errInvalidProtocolBufferSize, bufferSize)
	}

	ch := make(chan []byte, 1)

	select {
	case <-ctx.Done():
		close(ch)

		return ch, nil
	default:
	}

	go func() {
		defer close(ch)

		stopWatchingContext := watchContext(ctx, reader)
		defer stopWatchingContext()

		buffer := make([]byte, bufferSize)

		for {
			if ctx.Err() != nil {
				return
			}

			num, err := reader.Read(buffer)
			if num > 0 {
				data := make([]byte, num)
				copy(data, buffer[:num])

				if !send(ctx, ch, data) {
					return
				}
			}

			if err != nil {
				return
			}
		}
	}()

	return ch, nil
}

func send[T any](ctx context.Context, ch chan<- T, data T) bool {
	if ctx.Err() != nil {
		return false
	}

	select {
	case ch <- data:
		return true
	case <-ctx.Done():
		return false
	}
}

func watchContext(ctx context.Context, reader io.Reader) func() {
	ctxDone := make(chan struct{})

	go func() {
		select {
		case <-ctx.Done():
			if deadliner, ok := reader.(readDeadliner); ok {
				if err := deadliner.SetReadDeadline(time.Now()); err == nil {
					return
				}
			}

			if closer, ok := reader.(io.Closer); ok {
				_ = closer.Close()
			}
		case <-ctxDone:
		}
	}()

	return func() {
		close(ctxDone)
	}
}
