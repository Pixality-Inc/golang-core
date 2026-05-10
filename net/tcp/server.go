package tcp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"

	"github.com/pixality-inc/golang-core/logger"
	coreNet "github.com/pixality-inc/golang-core/net"
	internalServer "github.com/pixality-inc/golang-core/net/internal/server"
)

type Impl[T any] struct {
	log       logger.Loggable
	addr      string
	handler   coreNet.Handler[T]
	protocol  coreNet.Protocol[T]
	lifecycle *internalServer.Lifecycle[net.Listener]
	tlsConfig *tls.Config
}

func New[T any](addr string, handler coreNet.Handler[T], protocol coreNet.Protocol[T], opts ...Option) coreNet.Server[T] {
	serverOptions := &options{}
	for _, option := range opts {
		option(serverOptions)
	}

	server := &Impl[T]{
		log:       logger.NewLoggableImplWithService("tcp_server"),
		addr:      addr,
		handler:   handler,
		protocol:  protocol,
		lifecycle: internalServer.NewLifecycle[net.Listener](),
		tlsConfig: serverOptions.tlsConfig,
	}

	return server
}

func (s *Impl[T]) Start(ctx context.Context) error {
	log := s.log.GetLogger(ctx)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var listenConfig net.ListenConfig

	listener, err := listenConfig.Listen(ctx, "tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	if s.tlsConfig != nil {
		listener = tls.NewListener(listener, s.tlsConfig)
	}

	s.lifecycle.Set(listener, cancel)

	log.Infof("TCP server is listening on %s", s.addr)

	ctxDone := make(chan struct{})

	go func() {
		select {
		case <-ctx.Done():
			if sErr := s.shutdown(ctx); sErr != nil {
				s.log.GetLogger(ctx).WithError(sErr).Errorf("failed to shutdown TCP server")
			}
		case <-ctxDone:
		}
	}()

	if err = internalServer.AcceptStreamConnections(ctx, s.log, s.lifecycle, s.handler, s.protocol); err != nil {
		close(ctxDone)

		if sErr := s.shutdown(ctx); sErr != nil {
			log.WithError(sErr).Errorf("failed to shutdown TCP server")
		}

		s.lifecycle.Wait()

		return fmt.Errorf("accept connections: %w", err)
	}

	close(ctxDone)

	shutdownErr := s.shutdown(ctx)
	s.lifecycle.Wait()

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

func (s *Impl[T]) shutdown(ctx context.Context) error {
	return s.lifecycle.Shutdown(ctx, "listener")
}
