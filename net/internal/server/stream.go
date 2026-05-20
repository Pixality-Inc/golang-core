package server

import (
	"context"
	"fmt"
	"net"

	"github.com/pixality-inc/golang-core/logger"
	coreNet "github.com/pixality-inc/golang-core/net"
)

func AcceptStreamConnections[T any](
	ctx context.Context,
	loggable logger.Loggable,
	lifecycle *Lifecycle[net.Listener],
	handler coreNet.Handler[T],
	protocol coreNet.Protocol[T],
) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		listener, ok := lifecycle.Get()
		if !ok {
			return nil
		}

		netConnection, err := listener.Accept()
		if err != nil {
			if IsClosed(ctx, err) {
				return nil
			}

			return fmt.Errorf("accept: %w", err)
		}

		connection := coreNet.NewNetConnection(
			coreNet.NewConnectionId(),
			netConnection,
			protocol,
		)

		lifecycle.Go(func() {
			if handleErr := HandleStreamConnection(
				ctx,
				loggable.GetLogger(ctx),
				handler,
				protocol,
				netConnection,
				connection,
			); handleErr != nil {
				loggable.GetLogger(ctx).WithError(handleErr).Errorf("failed to handle connection")
			}
		})
	}
}

func HandleStreamConnection[T any](
	ctx context.Context,
	log logger.Logger,
	handler coreNet.Handler[T],
	protocol coreNet.Protocol[T],
	netConnection net.Conn,
	connection coreNet.Connection[T],
) error {
	stopWatchingContext := WatchContextClose(ctx, log, netConnection)
	defer stopWatchingContext()

	var client coreNet.Client[T]
	defer func() {
		CloseClient(log, client)
		Close(ctx, log, netConnection)
	}()

	client, err := OpenClient(ctx, handler, connection)
	if err != nil {
		return err
	}

	messages, err := protocol.Read(netConnection)
	if err != nil {
		return fmt.Errorf("read protocol: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case message, ok := <-messages:
			if !ok {
				return nil
			}

			if err = client.OnWrite(ctx, message); err != nil {
				log.WithError(err).Errorf("failed to read message")
			}
		}
	}
}
