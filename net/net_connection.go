package net

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/pixality-inc/golang-core/logger"
)

var ErrNetWrite = errors.New("write")

type NetConnectionImpl struct {
	log        logger.Loggable
	id         ConnectionId
	connection net.Conn
	address    Address
}

func NewNetConnection(
	id ConnectionId,
	connection net.Conn,
) Connection {
	return &NetConnectionImpl{
		log: logger.NewLoggableImplWithServiceAndFields("connection", logger.Fields{
			"id": id.String(),
		}),
		id:         id,
		connection: connection,
		address: NewAddress(
			connection.LocalAddr().String(),
			connection.RemoteAddr().String(),
		),
	}
}

func (c *NetConnectionImpl) Id() ConnectionId {
	return c.id
}

func (c *NetConnectionImpl) Address() Address {
	return c.address
}

func (c *NetConnectionImpl) Write(ctx context.Context, message Message) error {
	buffer := message.Data()

	num, err := c.connection.Write(buffer)
	if err != nil {
		return err
	}

	if num != len(buffer) {
		return fmt.Errorf("%w: wrote %d bytes, expected %d", ErrNetWrite, num, len(buffer))
	}

	return nil
}

func (c *NetConnectionImpl) Close() error {
	return c.connection.Close()
}
