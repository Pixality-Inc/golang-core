package net

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/pixality-inc/golang-core/logger"
	protocol2 "github.com/pixality-inc/golang-core/net/protocol"
)

var ErrNetWrite = errors.New("write")

type NetConnectionImpl[INP, OUT any] struct {
	log        logger.Loggable
	id         ConnectionId
	connection net.Conn
	address    Addresses
	protocol   protocol2.Protocol[INP, OUT]
}

func NewNetConnection[INP, OUT any](
	id ConnectionId,
	connection net.Conn,
	protocol protocol2.Protocol[INP, OUT],
) Connection[OUT] {
	return &NetConnectionImpl[INP, OUT]{
		log: logger.NewLoggableImplWithServiceAndFields("connection", logger.Fields{
			"id": id.String(),
		}),
		id:         id,
		connection: connection,
		address:    NewAddressesFromNet(connection.LocalAddr(), connection.RemoteAddr()),
		protocol:   protocol,
	}
}

func (c *NetConnectionImpl[INP, OUT]) Id() ConnectionId {
	return c.id
}

func (c *NetConnectionImpl[INP, OUT]) Address() Addresses {
	return c.address
}

func (c *NetConnectionImpl[INP, OUT]) Write(ctx context.Context, messages ...OUT) error {
	buffer, err := c.protocol.Marshal(ctx, messages...)
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

func (c *NetConnectionImpl[INP, OUT]) Close() error {
	return c.connection.Close()
}
