package net

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/pixality-inc/golang-core/logger"
)

var ErrNetWrite = errors.New("write")

type NetConnectionImpl[T any] struct {
	log        logger.Loggable
	id         ConnectionId
	connection net.Conn
	address    Addresses
	protocol   Protocol[T]
}

func NewNetConnection[T any](
	id ConnectionId,
	connection net.Conn,
	protocol Protocol[T],
) Connection[T] {
	return &NetConnectionImpl[T]{
		log: logger.NewLoggableImplWithServiceAndFields("connection", logger.Fields{
			"id": id.String(),
		}),
		id:         id,
		connection: connection,
		address:    NewAddressesFromNet(connection.LocalAddr(), connection.RemoteAddr()),
		protocol:   protocol,
	}
}

func (c *NetConnectionImpl[T]) Id() ConnectionId {
	return c.id
}

func (c *NetConnectionImpl[T]) Address() Addresses {
	return c.address
}

func (c *NetConnectionImpl[T]) Write(ctx context.Context, message T) error {
	buffer, err := c.protocol.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	num, err := c.connection.Write(buffer)
	if err != nil {
		return err
	}

	if num != len(buffer) {
		return fmt.Errorf("%w: wrote %d bytes, expected %d", ErrNetWrite, num, len(buffer))
	}

	return nil
}

func (c *NetConnectionImpl[T]) Close() error {
	return c.connection.Close()
}
