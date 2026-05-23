package websocket

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	fasthttpWebsocket "github.com/fasthttp/websocket"
	coreNet "github.com/pixality-inc/golang-core/net"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

type testClient struct {
	connection coreNet.Connection[[]byte]
	connected  chan struct{}
	closed     chan struct{}
	messages   chan []byte
	closeOnce  sync.Once
}

func newTestClient(connection coreNet.Connection[[]byte]) *testClient {
	return &testClient{
		connection: connection,
		connected:  make(chan struct{}),
		closed:     make(chan struct{}),
		messages:   make(chan []byte, 16),
	}
}

func (c *testClient) OnConnect(_ context.Context) error {
	close(c.connected)

	return nil
}

func (c *testClient) OnWrite(ctx context.Context, message []byte) error {
	data := make([]byte, len(message))
	copy(data, message)

	c.messages <- data

	return c.connection.Write(ctx, data)
}

func (c *testClient) OnClose(_ context.Context) error {
	c.closeOnce.Do(func() {
		close(c.closed)
	})

	return nil
}

type testHandler struct {
	clients   chan *testClient
	closed    chan struct{}
	closeOnce sync.Once
}

func newTestHandler() *testHandler {
	return &testHandler{
		clients: make(chan *testClient, 2),
		closed:  make(chan struct{}),
	}
}

func (h *testHandler) Handle(_ context.Context, connection coreNet.Connection[[]byte]) (coreNet.Client[[]byte], error) {
	client := newTestClient(connection)
	h.clients <- client

	return client, nil
}

func (h *testHandler) Close() error {
	h.closeOnce.Do(func() {
		close(h.closed)
	})

	return nil
}

func TestServerHandlesWebSocketMessages(t *testing.T) {
	t.Parallel()

	handler := newTestHandler()
	server := New(handler, coreNet.NewByteProtocol())

	listener, stopHTTPServer := startHTTPServer(t, server.Handler())
	defer stopHTTPServer()

	connection := dialWebSocket(t, listener)
	defer closeWebSocket(t, connection)

	client := waitClient(t, handler)
	waitClosed(t, client.connected)

	payload := []byte("hello websocket")
	require.NoError(t, connection.WriteMessage(fasthttpWebsocket.TextMessage, payload))
	require.Equal(t, payload, waitMessageData(t, client, len(payload)))

	messageType, data, err := connection.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, fasthttpWebsocket.BinaryMessage, messageType)
	require.Equal(t, payload, data)
}

func TestServerClosesClientsWhenFastHTTPShutsDown(t *testing.T) {
	t.Parallel()

	handler := newTestHandler()
	server := New(handler, coreNet.NewByteProtocol())

	listener, stopHTTPServer := startHTTPServer(t, server.Handler())
	defer stopHTTPServer()

	connection := dialWebSocket(t, listener)
	defer closeWebSocket(t, connection)

	client := waitClient(t, handler)
	waitClosed(t, client.connected)

	stopHTTPServer()
	waitClosed(t, client.closed)

	_, _, err := connection.ReadMessage()
	require.Error(t, err)
}

func TestServerRejectsNonUpgradeRequests(t *testing.T) {
	t.Parallel()

	handler := newTestHandler()
	server := New(handler, coreNet.NewByteProtocol())

	listener, stopHTTPServer := startHTTPServer(t, server.Handler())
	defer stopHTTPServer()

	client := fasthttp.Client{
		Dial: func(_ string) (net.Conn, error) {
			return listener.Dial()
		},
	}
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()

	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI("http://websocket.test")
	req.Header.SetMethod(fasthttp.MethodGet)

	require.NoError(t, client.Do(req, resp))
	require.Equal(t, fasthttp.StatusUpgradeRequired, resp.StatusCode())
}

func startHTTPServer(
	t *testing.T,
	handler fasthttp.RequestHandler,
) (*fasthttputil.InmemoryListener, func()) {
	t.Helper()

	listener := fasthttputil.NewInmemoryListener()

	server := &fasthttp.Server{
		Handler: handler,
	}

	serverErr := make(chan error, 1)

	go func() {
		serverErr <- server.Serve(listener)
	}()

	var stopOnce sync.Once

	return listener, func() {
		stopOnce.Do(func() {
			require.NoError(t, server.Shutdown())

			select {
			case err := <-serverErr:
				require.NoError(t, err)
			case <-time.After(time.Second):
				t.Fatal("timeout waiting for HTTP server shutdown")
			}
		})
	}
}

func dialWebSocket(
	t *testing.T,
	listener *fasthttputil.InmemoryListener,
) *fasthttpWebsocket.Conn {
	t.Helper()

	dialer := fasthttpWebsocket.Dialer{
		HandshakeTimeout: time.Second,
		NetDialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return listener.Dial()
		},
	}

	connection, response, err := dialer.DialContext(t.Context(), "ws://websocket.test", nil)
	if response != nil {
		defer func() {
			require.NoError(t, response.Body.Close())
		}()
	}

	require.NoError(t, err)
	require.Equal(t, http.StatusSwitchingProtocols, response.StatusCode)

	return connection
}

func waitClient(t *testing.T, handler *testHandler) *testClient {
	t.Helper()

	select {
	case client := <-handler.clients:
		return client
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for client")
	}

	return nil
}

func waitMessageData(t *testing.T, client *testClient, expectedLen int) []byte {
	t.Helper()

	deadline := time.After(time.Second)
	data := make([]byte, 0, expectedLen)

	for len(data) < expectedLen {
		select {
		case message := <-client.messages:
			data = append(data, message...)
		case <-deadline:
			t.Fatal("timeout waiting for message")
		}
	}

	return data
}

func waitClosed(t *testing.T, closed <-chan struct{}) {
	t.Helper()

	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for close")
	}
}

func closeWebSocket(t *testing.T, connection *fasthttpWebsocket.Conn) {
	t.Helper()

	if err := connection.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		require.NoError(t, err)
	}
}
