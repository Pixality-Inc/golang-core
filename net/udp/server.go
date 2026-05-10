package udp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/pixality-inc/golang-core/logger"
	coreNet "github.com/pixality-inc/golang-core/net"
	internalServer "github.com/pixality-inc/golang-core/net/internal/server"
)

var errUnsupportedPacketConnection = errors.New("unsupported packet connection")

type Impl[T any] struct {
	log       logger.Loggable
	addr      string
	handler   coreNet.Handler[T]
	protocol  coreNet.Protocol[T]
	lifecycle *internalServer.Lifecycle[*net.UDPConn]

	clients      map[string]clientSession[T]
	clientsMutex sync.Mutex
}

type clientSession[T any] struct {
	client coreNet.Client[T]
}

func New[T any](addr string, handler coreNet.Handler[T], protocol coreNet.Protocol[T]) coreNet.Server[T] {
	return &Impl[T]{
		log:       logger.NewLoggableImplWithService("udp_server"),
		addr:      addr,
		handler:   handler,
		protocol:  protocol,
		lifecycle: internalServer.NewLifecycle[*net.UDPConn](),

		clients:      make(map[string]clientSession[T]),
		clientsMutex: sync.Mutex{},
	}
}

func (s *Impl[T]) Start(ctx context.Context) error {
	log := s.log.GetLogger(ctx)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var listenConfig net.ListenConfig

	packetConnection, err := listenConfig.ListenPacket(ctx, "udp", s.addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	udpConnection, ok := packetConnection.(*net.UDPConn)
	if !ok {
		if cErr := packetConnection.Close(); cErr != nil {
			return fmt.Errorf("close unsupported packet connection: %w", cErr)
		}

		return fmt.Errorf("%w: %T", errUnsupportedPacketConnection, packetConnection)
	}

	s.lifecycle.Set(udpConnection, cancel)

	log.Infof("UDP server is listening on %s", udpConnection.LocalAddr().String())

	ctxDone := make(chan struct{})

	go func() {
		select {
		case <-ctx.Done():
			if sErr := s.shutdown(ctx); sErr != nil {
				s.log.GetLogger(ctx).WithError(sErr).Errorf("failed to shutdown UDP server")
			}
		case <-ctxDone:
		}
	}()

	if err = s.readMessages(ctx); err != nil {
		close(ctxDone)

		if sErr := s.shutdown(ctx); sErr != nil {
			log.WithError(sErr).Errorf("failed to shutdown UDP server")
		}

		s.lifecycle.Wait()
		s.closeClients(ctx)

		return fmt.Errorf("read messages: %w", err)
	}

	close(ctxDone)

	shutdownErr := s.shutdown(ctx)
	s.lifecycle.Wait()
	s.closeClients(ctx)

	if err = s.handler.Close(); err != nil {
		return fmt.Errorf("close handler: %w", err)
	}

	if shutdownErr != nil {
		return fmt.Errorf("shutdown: %w", shutdownErr)
	}

	return nil
}

func (s *Impl[T]) Stop() error {
	return s.shutdown(context.Background())
}

func (s *Impl[T]) readMessages(ctx context.Context) error {
	log := s.log.GetLogger(ctx)
	buffer := make([]byte, internalServer.ReadBufferSize)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		udpConnection, ok := s.lifecycle.Get()
		if !ok {
			return nil
		}

		num, remoteAddress, err := udpConnection.ReadFromUDP(buffer)
		if num > 0 && remoteAddress != nil {
			if hErr := s.handleDatagram(ctx, udpConnection, remoteAddress, buffer, num); hErr != nil {
				log.WithError(hErr).Errorf("failed to handle UDP datagram")
			}
		}

		if err != nil && internalServer.IsClosed(ctx, err) {
			break
		}

		if err != nil {
			return fmt.Errorf("read: %w", err)
		}
	}

	return nil
}

func (s *Impl[T]) handleDatagram(
	ctx context.Context,
	udpConnection *net.UDPConn,
	remoteAddress *net.UDPAddr,
	buffer []byte,
	num int,
) error {
	client, err := s.getClient(ctx, udpConnection, remoteAddress)
	if err != nil {
		return err
	}

	data := make([]byte, num)
	copy(data, buffer[:num])

	messages, err := s.protocol.Read(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("read protocol: %w", err)
	}

	for message := range messages {
		if err = client.OnWrite(ctx, message); err != nil {
			s.log.GetLogger(ctx).WithError(err).Errorf("failed to read message")
		}
	}

	return nil
}

func (s *Impl[T]) getClient(
	ctx context.Context,
	udpConnection *net.UDPConn,
	remoteAddress *net.UDPAddr,
) (coreNet.Client[T], error) {
	key := remoteAddress.String()

	s.clientsMutex.Lock()
	defer s.clientsMutex.Unlock()

	if session, ok := s.clients[key]; ok {
		return session.client, nil
	}

	connection := coreNet.NewPacketConnection(
		coreNet.NewConnectionId(),
		udpConnection,
		remoteAddress,
		s.protocol,
	)

	client, err := internalServer.OpenClient(ctx, s.handler, connection)
	if err != nil {
		return nil, err
	}

	s.clients[key] = clientSession[T]{
		client: client,
	}

	return client, nil
}

func (s *Impl[T]) closeClients(ctx context.Context) {
	s.clientsMutex.Lock()

	clients := make([]coreNet.Client[T], 0, len(s.clients))
	for _, session := range s.clients {
		clients = append(clients, session.client)
	}

	s.clients = make(map[string]clientSession[T])

	s.clientsMutex.Unlock()

	log := s.log.GetLogger(ctx)
	for _, client := range clients {
		internalServer.CloseClient(log, client)
	}
}

func (s *Impl[T]) shutdown(ctx context.Context) error {
	return s.lifecycle.Shutdown(ctx, "socket")
}
