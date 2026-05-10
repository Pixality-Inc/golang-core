package server

import (
	"context"
	"fmt"

	"github.com/pixality-inc/golang-core/logger"
	coreNet "github.com/pixality-inc/golang-core/net"
)

func WatchContextClose(ctx context.Context, log logger.Logger, closeable Closeable) func() {
	ctxDone := make(chan struct{})

	go func() {
		select {
		case <-ctx.Done():
			Close(ctx, log, closeable)
		case <-ctxDone:
		}
	}()

	return func() {
		close(ctxDone)
	}
}

func Close(ctx context.Context, log logger.Logger, closeable Closeable) {
	if err := closeable.Close(); err != nil && !IsClosed(ctx, err) {
		log.WithError(err).Errorf("failed to close connection")
	}
}

func OpenClient[T any](
	ctx context.Context,
	handler coreNet.Handler[T],
	connection coreNet.Connection[T],
) (coreNet.Client[T], error) {
	client, err := handler.Handle(ctx, connection)
	if err != nil {
		return nil, fmt.Errorf("failed to handle connection: %w", err)
	}

	if err = client.OnConnect(ctx); err != nil {
		return nil, fmt.Errorf("failed to handle OnConnect: %w", err)
	}

	return client, nil
}

func CloseClient[T any](log logger.Logger, client coreNet.Client[T]) {
	if client == nil {
		return
	}

	if err := client.OnClose(); err != nil {
		log.WithError(err).Errorf("failed to close client")
	}
}
