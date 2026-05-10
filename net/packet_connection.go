package net

import (
	"context"
	"fmt"
	stdNet "net"
)

type PacketConnectionImpl[T any] struct {
	id            ConnectionId
	connection    stdNet.PacketConn
	remoteAddress stdNet.Addr
	address       Addresses
	protocol      Protocol[T]
}

func NewPacketConnection[T any](
	id ConnectionId,
	connection stdNet.PacketConn,
	remoteAddress stdNet.Addr,
	protocol Protocol[T],
) Connection[T] {
	return &PacketConnectionImpl[T]{
		id:            id,
		connection:    connection,
		remoteAddress: remoteAddress,
		address:       NewAddressesFromNet(connection.LocalAddr(), remoteAddress),
		protocol:      protocol,
	}
}

func (c *PacketConnectionImpl[T]) Id() ConnectionId {
	return c.id
}

func (c *PacketConnectionImpl[T]) Address() Addresses {
	return c.address
}

func (c *PacketConnectionImpl[T]) Write(_ context.Context, message T) error {
	buffer, err := c.protocol.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	num, err := c.connection.WriteTo(buffer, c.remoteAddress)
	if err != nil {
		return err
	}

	if num != len(buffer) {
		return fmt.Errorf("%w: wrote %d bytes, expected %d", ErrNetWrite, num, len(buffer))
	}

	return nil
}

func (c *PacketConnectionImpl[T]) Close() error {
	return nil
}
