package server

import (
	"context"
	"fmt"
	"net"

	"github.com/pixality-inc/golang-core/logger"
	coreNet "github.com/pixality-inc/golang-core/net"
	"github.com/pixality-inc/golang-core/net/protocol"
)

func AcceptStreamConnections[INP, OUT any](
	ctx context.Context,
	loggable logger.Loggable,
	lifecycle *Lifecycle[net.Listener],
	handler coreNet.Handler[INP, OUT],
	protocol protocol.Protocol[INP, OUT],
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

func HandleStreamConnection[INP, OUT any](
	ctx context.Context,
	log logger.Logger,
	handler coreNet.Handler[INP, OUT],
	protocol protocol.Protocol[INP, OUT],
	netConnection net.Conn,
	connection coreNet.Connection[OUT],
) error {
	stopWatchingContext := WatchContextClose(ctx, log, netConnection)
	defer stopWatchingContext()

	var client coreNet.Client[INP]

	defer func() {
		CloseClient(ctx, log, client)

		Close(ctx, log, netConnection)
	}()

	client, err := OpenClient(ctx, handler, connection)
	if err != nil {
		return err
	}

	messages, err := protocol.Read(ctx, netConnection)
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
