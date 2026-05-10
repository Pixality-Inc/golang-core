package udp

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	coreNet "github.com/pixality-inc/golang-core/net"
	"github.com/stretchr/testify/require"
)

type testClient struct {
	closed    chan struct{}
	messages  chan []byte
	closeOnce sync.Once
}

func newTestClient() *testClient {
	return &testClient{
		closed:   make(chan struct{}),
		messages: make(chan []byte, 16),
	}
}

func (c *testClient) OnConnect(_ context.Context) error {
	return nil
}

func (c *testClient) OnWrite(_ context.Context, message []byte) error {
	data := make([]byte, len(message))
	copy(data, message)

	c.messages <- data

	return nil
}

func (c *testClient) OnClose() error {
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

func (h *testHandler) Handle(_ context.Context, _ coreNet.Connection[[]byte]) (coreNet.Client[[]byte], error) {
	client := newTestClient()
	h.clients <- client

	return client, nil
}

func (h *testHandler) Close() error {
	h.closeOnce.Do(func() {
		close(h.closed)
	})

	return nil
}

func TestServerReadsDatagram(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	handler := newTestHandler()
	server, ok := New(net.JoinHostPort("127.0.0.1", "0"), handler, coreNet.NewByteProtocol()).(*Impl[[]byte])
	require.True(t, ok)

	serverErr := make(chan error, 1)

	go func() {
		serverErr <- server.Start(ctx)
	}()

	connection := dialUDP(t, waitServerAddress(t, server))
	defer closeConnection(t, connection)

	payload := []byte("udp message")

	num, err := connection.Write(payload)
	require.NoError(t, err)
	require.Equal(t, len(payload), num)

	client := waitClient(t, handler)
	require.Equal(t, payload, waitMessageData(t, client, len(payload)))

	cancel()
	require.NoError(t, waitServer(t, serverErr))
}

func TestServerClosesClientsWhenContextDone(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	handler := newTestHandler()
	server, ok := New(net.JoinHostPort("127.0.0.1", "0"), handler, coreNet.NewByteProtocol()).(*Impl[[]byte])
	require.True(t, ok)

	serverErr := make(chan error, 1)

	go func() {
		serverErr <- server.Start(ctx)
	}()

	addr := waitServerAddress(t, server)

	connection1 := dialUDP(t, addr)
	defer closeConnection(t, connection1)

	connection2 := dialUDP(t, addr)
	defer closeConnection(t, connection2)

	_, err := connection1.Write([]byte("client 1"))
	require.NoError(t, err)

	_, err = connection2.Write([]byte("client 2"))
	require.NoError(t, err)

	client1 := waitClient(t, handler)
	client2 := waitClient(t, handler)

	cancel()

	require.NoError(t, waitServer(t, serverErr))
	waitClosed(t, client1.closed)
	waitClosed(t, client2.closed)
	waitClosed(t, handler.closed)
}

func waitServerAddress(t *testing.T, server *Impl[[]byte]) string {
	t.Helper()

	var addr string

	require.Eventually(t, func() bool {
		udpConnection, ok := server.lifecycle.Get()
		if !ok {
			return false
		}

		addr = udpConnection.LocalAddr().String()

		return true
	}, time.Second, 10*time.Millisecond)

	return addr
}

func dialUDP(t *testing.T, addr string) net.Conn {
	t.Helper()

	dialer := net.Dialer{
		Timeout: time.Second,
	}

	connection, err := dialer.DialContext(t.Context(), "udp", addr)
	require.NoError(t, err)

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

func waitServer(t *testing.T, serverErr <-chan error) error {
	t.Helper()

	select {
	case err := <-serverErr:
		return err
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for server shutdown")
	}

	return nil
}

func waitClosed(t *testing.T, closed <-chan struct{}) {
	t.Helper()

	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for close")
	}
}

func closeConnection(t *testing.T, connection net.Conn) {
	t.Helper()

	if err := connection.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		require.NoError(t, err)
	}
}
