package websocket

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	fasthttpWebsocket "github.com/fasthttp/websocket"
	coreNet "github.com/pixality-inc/golang-core/net"
	protocol2 "github.com/pixality-inc/golang-core/net/protocol"
)

// close codes of CloseWithCode, an application may use its own ones in 4000-4999
const (
	CloseNormalClosure     = fasthttpWebsocket.CloseNormalClosure
	CloseGoingAway         = fasthttpWebsocket.CloseGoingAway
	ClosePolicyViolation   = fasthttpWebsocket.ClosePolicyViolation
	CloseMessageTooBig     = fasthttpWebsocket.CloseMessageTooBig
	CloseInternalServerErr = fasthttpWebsocket.CloseInternalServerErr
	CloseServiceRestart    = fasthttpWebsocket.CloseServiceRestart
	CloseTryAgainLater     = fasthttpWebsocket.CloseTryAgainLater
)

// closeFrameWriteTimeout bounds the write of the close frame to a peer that does not read
const closeFrameWriteTimeout = time.Second

// Connection is the websocket connection a handler gets (as net.Connection)
type Connection[OUT any] interface {
	coreNet.Connection[OUT]

	// CloseWithCode sends a close frame with the code and the reason (at most 123 bytes) and closes
	// the connection: the peer sees the code in its close event
	CloseWithCode(code int, reason string) error
}

type ConnectionImpl[INP, OUT any] struct {
	id               coreNet.ConnectionId
	connection       *fasthttpWebsocket.Conn
	netConnection    net.Conn
	address          coreNet.Addresses
	protocol         protocol2.Protocol[INP, OUT]
	writeMessageType int
	mutex            sync.Mutex
}

func NewConnection[INP, OUT any](
	id coreNet.ConnectionId,
	connection *fasthttpWebsocket.Conn,
	netConnection net.Conn,
	protocol protocol2.Protocol[INP, OUT],
	writeMessageType int,
) Connection[OUT] {
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

func (c *ConnectionImpl[INP, OUT]) Write(ctx context.Context, messages ...OUT) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if deadline, ok := ctx.Deadline(); ok {
		if err := c.connection.SetWriteDeadline(deadline); err != nil {
			return fmt.Errorf("set write deadline: %w", err)
		}
	}

	data, err := c.protocol.Marshal(ctx, messages...)
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

func (c *ConnectionImpl[INP, OUT]) CloseWithCode(code int, reason string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	writeErr := c.connection.WriteControl(
		fasthttpWebsocket.CloseMessage,
		fasthttpWebsocket.FormatCloseMessage(code, reason),
		time.Now().Add(closeFrameWriteTimeout),
	)
	if errors.Is(writeErr, fasthttpWebsocket.ErrCloseSent) {
		writeErr = nil
	}

	closeErr := closeConnection(c.connection, c.netConnection)
	if errors.Is(closeErr, net.ErrClosed) {
		closeErr = nil
	}

	if writeErr != nil {
		return fmt.Errorf("write close frame: %w", errors.Join(writeErr, closeErr))
	}

	return closeErr
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
