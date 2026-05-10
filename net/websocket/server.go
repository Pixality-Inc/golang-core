package websocket

import (
	"context"
	"fmt"
	"io"
	"net"

	fasthttpWebsocket "github.com/fasthttp/websocket"
	"github.com/pixality-inc/golang-core/logger"
	coreNet "github.com/pixality-inc/golang-core/net"
	internalServer "github.com/pixality-inc/golang-core/net/internal/server"
	"github.com/valyala/fasthttp"
)

type Server[T any] interface {
	Handler() fasthttp.RequestHandler
	Handle(ctx *fasthttp.RequestCtx)
}

type Impl[T any] struct {
	log              logger.Loggable
	handler          coreNet.Handler[T]
	protocol         coreNet.Protocol[T]
	upgrader         fasthttpWebsocket.FastHTTPUpgrader
	writeMessageType int
}

func New[T any](handler coreNet.Handler[T], protocol coreNet.Protocol[T], opts ...Option) Server[T] {
	serverOptions := defaultOptions()
	for _, option := range opts {
		option(serverOptions)
	}

	server := &Impl[T]{
		log:              logger.NewLoggableImplWithService("websocket_server"),
		handler:          handler,
		protocol:         protocol,
		upgrader:         serverOptions.upgrader,
		writeMessageType: serverOptions.writeMessageType,
	}

	return server
}

func NewRequestHandler[T any](handler coreNet.Handler[T], protocol coreNet.Protocol[T], options ...Option) fasthttp.RequestHandler {
	return New(handler, protocol, options...).Handler()
}

func (s *Impl[T]) Handler() fasthttp.RequestHandler {
	return s.Handle
}

func (s *Impl[T]) Handle(ctx *fasthttp.RequestCtx) {
	log := s.log.GetLogger(ctx)

	if !fasthttpWebsocket.FastHTTPIsWebSocketUpgrade(ctx) {
		ctx.Error("websocket upgrade required", fasthttp.StatusUpgradeRequired)

		return
	}

	done := ctx.Done()
	netConnection := ctx.Conn()

	if err := s.upgrader.Upgrade(ctx, func(connection *fasthttpWebsocket.Conn) {
		s.handleConnection(done, connection, netConnection)
	}); err != nil {
		log.WithError(err).Errorf("failed to upgrade websocket connection")
	}
}

func (s *Impl[T]) handleConnection(
	done <-chan struct{},
	websocketConnection *fasthttpWebsocket.Conn,
	netConnection net.Conn,
) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := s.log.GetLogger(ctx)

	stopWatchingRequest := s.watchDone(ctx, done, websocketConnection, netConnection, cancel)
	defer stopWatchingRequest()

	connection := NewConnection(
		coreNet.NewConnectionId(),
		websocketConnection,
		netConnection,
		s.protocol,
		s.writeMessageType,
	)

	var client coreNet.Client[T]
	defer func() {
		internalServer.CloseClient(log, client)
		internalServer.Close(ctx, log, connection)
	}()

	var err error

	client, err = internalServer.OpenClient(ctx, s.handler, connection)
	if err != nil {
		log.WithError(err).Errorf("failed to open websocket client")

		return
	}

	for {
		_, reader, readErr := websocketConnection.NextReader()
		if readErr != nil {
			if isExpectedClose(readErr) || internalServer.IsClosed(ctx, readErr) {
				return
			}

			log.WithError(readErr).Errorf("failed to read websocket message")

			return
		}

		if err = s.handleMessages(ctx, client, reader); err != nil {
			log.WithError(err).Errorf("failed to read websocket message")

			return
		}
	}
}

func (s *Impl[T]) handleMessages(ctx context.Context, client coreNet.Client[T], reader io.Reader) error {
	messages, err := s.protocol.Read(reader)
	if err != nil {
		return fmt.Errorf("read protocol: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case message, ok := <-messages:
			if !ok {
				return nil
			}

			if err = client.OnWrite(ctx, message); err != nil {
				s.log.GetLogger(ctx).WithError(err).Errorf("failed to read message")
			}
		}
	}
}

func (s *Impl[T]) watchDone(
	ctx context.Context,
	done <-chan struct{},
	connection *fasthttpWebsocket.Conn,
	netConnection net.Conn,
	cancel context.CancelFunc,
) func() {
	if done == nil {
		return func() {}
	}

	ctxDone := make(chan struct{})

	go func() {
		select {
		case <-done:
			cancel()

			if err := closeConnection(connection, netConnection); err != nil && !internalServer.IsClosed(ctx, err) {
				s.log.GetLogger(ctx).WithError(err).Errorf("failed to close websocket connection")
			}
		case <-ctxDone:
		}
	}()

	return func() {
		close(ctxDone)
	}
}

func isExpectedClose(err error) bool {
	return fasthttpWebsocket.IsCloseError(
		err,
		fasthttpWebsocket.CloseNormalClosure,
		fasthttpWebsocket.CloseGoingAway,
		fasthttpWebsocket.CloseNoStatusReceived,
		fasthttpWebsocket.CloseAbnormalClosure,
	)
}
