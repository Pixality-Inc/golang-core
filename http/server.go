package http

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/pixality-inc/golang-core/logger"

	"github.com/valyala/fasthttp"
)

type Server interface {
	Name() string
	ListenAndServe(ctx context.Context) error
	Stop() error
}

type Impl struct {
	log             logger.Loggable
	name            string
	bindingAddress  string
	shutdownTimeout time.Duration
	handler         fasthttp.RequestHandler
	httpServer      *fasthttp.Server
	connected       bool
	mutex           sync.Mutex
}

func New(
	name string,
	cfg Config,
	handler fasthttp.RequestHandler,
) Server {
	return &Impl{
		log: logger.NewLoggableImplWithServiceAndFields(
			"http_server",
			logger.Fields{
				"name": name,
			},
		),
		name:            name,
		bindingAddress:  cfg.Address(),
		shutdownTimeout: cfg.ShutdownTimeout(),
		handler:         handler,
		httpServer:      &fasthttp.Server{},
		connected:       false,
		mutex:           sync.Mutex{},
	}
}

func (s *Impl) Name() string {
	return s.name
}

func (s *Impl) ListenAndServe(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.connected = true

	log := s.log.GetLogger(ctx)

	log.Infof("Binding to %s...", s.bindingAddress)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	channel := make(chan error)

	go func() {
		channel <- s.serve()
	}()

	select {
	case <-ctx.Done():
		err := ctx.Err()

		if !errors.Is(err, context.Canceled) {
			return err
		}

		return nil

	case err := <-channel:
		return err
	}
}

func (s *Impl) Stop() error {
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer shutdownCancel()

	return s.shutdown(shutdownCtx)
}

func (s *Impl) shutdown(ctx context.Context) error {
	if !s.connected {
		return nil
	}

	log := s.log.GetLogger(ctx)

	log.Info("Shutting down HTTP server...")

	if err := s.httpServer.ShutdownWithContext(ctx); err != nil {
		return err
	}

	log.Info("HTTP server shut down successfully")

	return nil
}

func (s *Impl) serve() error {
	s.httpServer.Handler = s.handler
	s.httpServer.MaxRequestBodySize = 256 * 1024 * 1024

	return s.httpServer.ListenAndServe(s.bindingAddress)
}
