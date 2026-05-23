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

type Server[INP, OUT any] interface {
	Handler() fasthttp.RequestHandler
	Handle(ctx *fasthttp.RequestCtx)
}

type Impl[INP, OUT any] struct {
	log              logger.Loggable
	handler          coreNet.Handler[INP, OUT]
	protocol         coreNet.Protocol[INP, OUT]
	upgrader         fasthttpWebsocket.FastHTTPUpgrader
	writeMessageType int
}

func New[INP, OUT any](
	handler coreNet.Handler[INP, OUT],
	protocol coreNet.Protocol[INP, OUT],
	opts ...Option,
) Server[INP, OUT] {
	serverOptions := defaultOptions()
	for _, option := range opts {
		option(serverOptions)
	}

	server := &Impl[INP, OUT]{
		log:              logger.NewLoggableImplWithService("websocket_server"),
		handler:          handler,
		protocol:         protocol,
		upgrader:         serverOptions.upgrader,
		writeMessageType: serverOptions.writeMessageType,
	}

	return server
}

func NewRequestHandler[INP, OUT any](
	handler coreNet.Handler[INP, OUT],
	protocol coreNet.Protocol[INP, OUT],
	options ...Option,
) fasthttp.RequestHandler {
	return New(handler, protocol, options...).Handler()
}

func (s *Impl[INP, OUT]) Handler() fasthttp.RequestHandler {
	return s.Handle
}

func (s *Impl[INP, OUT]) Handle(ctx *fasthttp.RequestCtx) {
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

func (s *Impl[INP, OUT]) handleConnection(
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

	var client coreNet.Client[INP]

	defer func() {
		internalServer.CloseClient(ctx, log, client)

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

func (s *Impl[INP, OUT]) handleMessages(ctx context.Context, client coreNet.Client[INP], reader io.Reader) error {
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

func (s *Impl[INP, OUT]) watchDone(
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
