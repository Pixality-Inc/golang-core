package websocket

import (
	"context"
	"fmt"
	"net"
	"sync"

	fasthttpWebsocket "github.com/fasthttp/websocket"
	coreNet "github.com/pixality-inc/golang-core/net"
)

type ConnectionImpl[INP, OUT any] struct {
	id               coreNet.ConnectionId
	connection       *fasthttpWebsocket.Conn
	netConnection    net.Conn
	address          coreNet.Addresses
	protocol         coreNet.Protocol[INP, OUT]
	writeMessageType int
	mutex            sync.Mutex
}

func NewConnection[INP, OUT any](
	id coreNet.ConnectionId,
	connection *fasthttpWebsocket.Conn,
	netConnection net.Conn,
	protocol coreNet.Protocol[INP, OUT],
	writeMessageType int,
) coreNet.Connection[OUT] {
	return &ConnectionImpl[INP, OUT]{
		id:               id,
		connection:       connection,
		netConnection:    netConnection,
		address:          coreNet.NewAddressesFromNet(connection.LocalAddr(), connection.RemoteAddr()),
		protocol:         protocol,
		writeMessageType: writeMessageType,
		mutex:            sync.Mutex{},
	}
}

func (c *ConnectionImpl[INP, OUT]) Id() coreNet.ConnectionId {
	return c.id
}

func (c *ConnectionImpl[INP, OUT]) Address() coreNet.Addresses {
	return c.address
}

func (c *ConnectionImpl[INP, OUT]) Write(ctx context.Context, message OUT) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if deadline, ok := ctx.Deadline(); ok {
		if err := c.connection.SetWriteDeadline(deadline); err != nil {
			return fmt.Errorf("set write deadline: %w", err)
		}
	}

	data, err := c.protocol.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	if err = c.connection.WriteMessage(c.writeMessageType, data); err != nil {
		return err
	}

	return nil
}

func (c *ConnectionImpl[INP, OUT]) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return closeConnection(c.connection, c.netConnection)
}

func closeConnection(connection *fasthttpWebsocket.Conn, netConnection net.Conn) error {
	if netConnection == nil {
		if connection == nil {
			return net.ErrClosed
		}

		netConnection = connection.NetConn()
	}

	if netConnection != nil {
		return netConnection.Close()
	}

	return net.ErrClosed
}
