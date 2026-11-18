package websocket

import (
	"testing"
	"time"

	fasthttpWebsocket "github.com/fasthttp/websocket"
	"github.com/stretchr/testify/require"

	"github.com/pixality-inc/golang-core/net/protocol"
)

const roomFullCloseCode = 4429

func TestServerClosesWithCode(t *testing.T) {
	t.Parallel()

	handler := newTestHandler()
	server := New(handler, protocol.NewBinary())

	listener, stopHTTPServer := startHTTPServer(t, server.Handler())
	defer stopHTTPServer()

	connection := dialWebSocket(t, listener)
	defer closeWebSocket(t, connection)

	client := waitClient(t, handler)
	waitClosed(t, client.connected)

	websocketConnection, ok := client.connection.(Connection[[]byte])
	require.True(t, ok)
	require.NoError(t, websocketConnection.CloseWithCode(roomFullCloseCode, "room-full"))

	_, _, err := connection.ReadMessage()
	require.True(t, fasthttpWebsocket.IsCloseError(err, roomFullCloseCode), "unexpected error: %v", err)

	var closeErr *fasthttpWebsocket.CloseError
	require.ErrorAs(t, err, &closeErr)
	require.Equal(t, "room-full", closeErr.Text)

	waitClosed(t, client.closed)
}

func TestServerClosesOnReadLimit(t *testing.T) {
	t.Parallel()

	handler := newTestHandler()
	server := New(handler, protocol.NewBinary(), WithReadLimit(8))

	listener, stopHTTPServer := startHTTPServer(t, server.Handler())
	defer stopHTTPServer()

	connection := dialWebSocket(t, listener)
	defer closeWebSocket(t, connection)

	client := waitClient(t, handler)
	waitClosed(t, client.connected)

	require.NoError(t, connection.WriteMessage(fasthttpWebsocket.TextMessage, []byte("longer than eight bytes")))

	_, _, err := connection.ReadMessage()
	require.True(t, fasthttpWebsocket.IsCloseError(err, CloseMessageTooBig), "unexpected error: %v", err)

	waitClosed(t, client.closed)
}

func TestServerKeepAliveClosesSilentPeer(t *testing.T) {
	t.Parallel()

	handler := newTestHandler()
	server := New(handler, protocol.NewBinary(), WithKeepAlive(20*time.Millisecond, 60*time.Millisecond))

	listener, stopHTTPServer := startHTTPServer(t, server.Handler())
	defer stopHTTPServer()

	connection := dialWebSocket(t, listener)
	defer closeWebSocket(t, connection)

	connection.SetPingHandler(func(string) error { return nil })

	go func() {
		for {
			if _, _, err := connection.ReadMessage(); err != nil {
				return
			}
		}
	}()

	client := waitClient(t, handler)
	waitClosed(t, client.connected)
	waitClosed(t, client.closed)
}

func TestServerKeepAliveKeepsAnsweringPeer(t *testing.T) {
	t.Parallel()

	handler := newTestHandler()
	server := New(handler, protocol.NewBinary(), WithKeepAlive(20*time.Millisecond, 60*time.Millisecond))

	listener, stopHTTPServer := startHTTPServer(t, server.Handler())
	defer stopHTTPServer()

	connection := dialWebSocket(t, listener)
	defer closeWebSocket(t, connection)

	go func() {
		for {
			if _, _, err := connection.ReadMessage(); err != nil {
				return
			}
		}
	}()

	client := waitClient(t, handler)
	waitClosed(t, client.connected)

	select {
	case <-client.closed:
		t.Fatal("a peer answering the pings was closed")
	case <-time.After(300 * time.Millisecond):
	}
}

func TestServerWritesTextMessages(t *testing.T) {
	t.Parallel()

	handler := newTestHandler()
	server := New(handler, protocol.NewBinary(), WithWriteMessageType(TextMessage))

	listener, stopHTTPServer := startHTTPServer(t, server.Handler())
	defer stopHTTPServer()

	connection := dialWebSocket(t, listener)
	defer closeWebSocket(t, connection)

	client := waitClient(t, handler)
	waitClosed(t, client.connected)

	payload := []byte(`{"type":"hello"}`)
	require.NoError(t, connection.WriteMessage(fasthttpWebsocket.TextMessage, payload))

	messageType, data, err := connection.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, fasthttpWebsocket.TextMessage, messageType)
	require.Equal(t, payload, data)
}
