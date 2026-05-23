package unix

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/pixality-inc/golang-core/logger"
	coreNet "github.com/pixality-inc/golang-core/net"
	internalServer "github.com/pixality-inc/golang-core/net/internal/server"
	protocol2 "github.com/pixality-inc/golang-core/net/protocol"
)

var errSocketPathExists = errors.New("socket path exists and is not a unix socket")

type Impl[INP, OUT any] struct {
	log       logger.Loggable
	addr      string
	handler   coreNet.Handler[INP, OUT]
	protocol  protocol2.Protocol[INP, OUT]
	lifecycle *internalServer.Lifecycle[net.Listener]
}

func New[INP, OUT any](
	addr string,
	handler coreNet.Handler[INP, OUT],
	protocol protocol2.Protocol[INP, OUT],
) coreNet.Server[INP, OUT] {
	return &Impl[INP, OUT]{
		log:       logger.NewLoggableImplWithService("unix_server"),
		addr:      addr,
		handler:   handler,
		protocol:  protocol,
		lifecycle: internalServer.NewLifecycle[net.Listener](),
	}
}

func (s *Impl[INP, OUT]) Start(ctx context.Context) error {
	log := s.log.GetLogger(ctx)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if err := s.removeSocketFile(); err != nil {
		return fmt.Errorf("remove stale socket file: %w", err)
	}

	var listenConfig net.ListenConfig

	listener, err := listenConfig.Listen(ctx, "unix", s.addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	s.lifecycle.Set(listener, cancel)

	log.Infof("Unix socket server is listening on %s", s.addr)

	ctxDone := make(chan struct{})

	go func() {
		select {
		case <-ctx.Done():
			if sErr := s.shutdown(ctx); sErr != nil {
				s.log.GetLogger(ctx).WithError(sErr).Errorf("failed to shutdown Unix socket server")
			}
		case <-ctxDone:
		}
	}()

	if err = internalServer.AcceptStreamConnections(ctx, s.log, s.lifecycle, s.handler, s.protocol); err != nil {
		close(ctxDone)

		if sErr := s.shutdown(ctx); sErr != nil {
			log.WithError(sErr).Errorf("failed to shutdown Unix socket server")
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

func (s *Impl[INP, OUT]) Stop() error {
	return s.shutdown(context.Background())
}

func (s *Impl[INP, OUT]) shutdown(ctx context.Context) error {
	shutdownErr := s.lifecycle.Shutdown(ctx, "listener")
	removeErr := s.removeSocketFile()

	return errors.Join(shutdownErr, removeErr)
}

func (s *Impl[INP, OUT]) removeSocketFile() error {
	fileInfo, err := os.Lstat(s.addr)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	if err != nil {
		return err
	}

	if fileInfo.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("%w: %s", errSocketPathExists, s.addr)
	}

	if err = os.Remove(s.addr); err != nil {
		return fmt.Errorf("remove socket file: %w", err)
	}

	return nil
}
