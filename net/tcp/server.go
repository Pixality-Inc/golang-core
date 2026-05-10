package tcp

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/pixality-inc/golang-core/logger"
	coreNet "github.com/pixality-inc/golang-core/net"
	"github.com/pixality-inc/golang-core/util"
)

type Impl struct {
	log      logger.Loggable
	addr     string
	handler  coreNet.Handler
	listener net.Listener
}

func New(addr string, handler coreNet.Handler) coreNet.Server {
	return &Impl{
		log:      logger.NewLoggableImplWithService("tcp_server"),
		addr:     addr,
		handler:  handler,
		listener: nil,
	}
}

func (s *Impl) Start(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	defer func() {
		if fErr := listener.Close(); fErr != nil {
			s.log.GetLogger(ctx).WithError(fErr).Errorf("failed to close listener")
		}
	}()

	s.listener = listener

	if err = s.acceptConnections(ctx); err != nil {
		return fmt.Errorf("accept connections: %w", err)
	}

	return nil
}

func (s *Impl) Stop() error {
	// @todo!!!
	return util.ErrNotImplemented
}

func (s *Impl) acceptConnections(ctx context.Context) error {
	for {
		netConnection, err := s.listener.Accept()
		if err != nil {
			return fmt.Errorf("accept: %w", err)
		}

		connectionId := coreNet.NewConnectionId()

		connection := coreNet.NewNetConnection(connectionId, netConnection)

		go func() {
			if err = s.handleConnection(ctx, netConnection, connection); err != nil {
				s.log.GetLogger(ctx).WithError(err).Errorf("failed to handle connection")
			}
		}()
	}
}

func (s *Impl) handleConnection(
	ctx context.Context,
	netConnection net.Conn,
	connection coreNet.Connection,
) error {
	log := s.log.GetLogger(ctx)

	client, err := s.handler.Handle(ctx, connection)
	if err != nil {
		return fmt.Errorf("failed to handle connection: %w", err)
	}

	defer func() {
		if err = client.OnClose(); err != nil {
			log.WithError(err).Errorf("failed to close client")
		}

		if err = netConnection.Close(); err != nil {
			log.WithError(err).Errorf("failed to close connection")
		}
	}()

	for {
		reader := bufio.NewReader(netConnection)
		data, err := reader.ReadBytes(13)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}

			return fmt.Errorf("read: %w", err)
		}

		message := coreNet.NewMessage(data)

		if err = client.OnWrite(ctx, message); err != nil {
			log.WithError(err).Errorf("failed to read message")
		}
	}
}
