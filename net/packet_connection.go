package net

import (
	"context"
	"fmt"
	stdNet "net"
)

type PacketConnectionImpl[INP, OUT any] struct {
	id            ConnectionId
	connection    stdNet.PacketConn
	remoteAddress stdNet.Addr
	address       Addresses
	protocol      Protocol[INP, OUT]
}

func NewPacketConnection[INP, OUT any](
	id ConnectionId,
	connection stdNet.PacketConn,
	remoteAddress stdNet.Addr,
	protocol Protocol[INP, OUT],
) Connection[OUT] {
	return &PacketConnectionImpl[INP, OUT]{
		id:            id,
		connection:    connection,
		remoteAddress: remoteAddress,
		address:       NewAddressesFromNet(connection.LocalAddr(), remoteAddress),
		protocol:      protocol,
	}
}

func (c *PacketConnectionImpl[INP, OUT]) Id() ConnectionId {
	return c.id
}

func (c *PacketConnectionImpl[INP, OUT]) Address() Addresses {
	return c.address
}

func (c *PacketConnectionImpl[INP, OUT]) Write(_ context.Context, message OUT) error {
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

func (c *PacketConnectionImpl[INP, OUT]) Close() error {
	return nil
}
