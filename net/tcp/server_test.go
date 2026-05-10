package tcp

import (
	"context"
	"fmt"
	"net"
	"testing"

	coreNet "github.com/pixality-inc/golang-core/net"
	"github.com/stretchr/testify/require"
)

type myClient struct {
	connection coreNet.Connection
}

func NewMyClient(connection coreNet.Connection) coreNet.Client {
	return &myClient{
		connection: connection,
	}
}

func (c *myClient) OnWrite(ctx context.Context, message coreNet.Message) error {
	fmt.Printf("[CLIENT READ] %#v\n", message)

	return nil
}

func (c *myClient) OnClose() error {
	fmt.Println("[CLIENT CLOSE]")

	return nil
}

type myHandler struct{}

func NewMyHandler() coreNet.Handler {
	return &myHandler{}
}

func (h *myHandler) Handle(ctx context.Context, connection coreNet.Connection) (coreNet.Client, error) {
	return NewMyClient(connection), nil
}

func (h *myHandler) Close() error {
	return nil
}

func Test_Server(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	handler := NewMyHandler()

	srv := New(net.JoinHostPort("127.0.0.1", "5000"), handler)

	err := srv.Start(ctx)
	require.NoError(t, err)
}
