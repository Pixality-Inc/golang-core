package control_flow

import (
	"context"
	"os"
	"os/signal"
	"sync"

	"github.com/pixality-inc/golang-core/logger"
)

type Shutdown interface {
	Stop() error
}

type ShutdownWithName interface {
	Shutdown

	Name() string
}

type ControlFlow interface {
	RegisterClosableService(name string, closer Closable)
	RegisterClosableWithErrorService(name string, closer ClosableWithError)
	RegisterStoppableService(name string, stoppable Stoppable)
	RegisterShutdownService(name string, service Shutdown)
	RegisterShutdownServiceWithName(service ShutdownWithName)
	Context() context.Context
	Cancel() context.CancelFunc
	WaitForInterrupt()
	Shutdown()
}

//nolint:containedctx
type ControlFlowImpl struct {
	log      logger.Loggable
	context  context.Context
	cancel   context.CancelFunc
	services map[string]Shutdown

	mutex sync.Mutex
}

func NewControlFlow(ctx context.Context) *ControlFlowImpl {
	ctx, cancel := context.WithCancel(ctx)

	return &ControlFlowImpl{
		log:      logger.NewLoggableImplWithService("control_flow"),
		context:  ctx,
		cancel:   cancel,
		services: make(map[string]Shutdown),
		mutex:    sync.Mutex{},
	}
}

func (c *ControlFlowImpl) RegisterClosableService(name string, closer Closable) {
	c.RegisterShutdownService(name, NewClosable(closer))
}

func (c *ControlFlowImpl) RegisterClosableWithErrorService(name string, closer ClosableWithError) {
	c.RegisterShutdownService(name, NewClosableWithError(closer))
}

func (c *ControlFlowImpl) RegisterStoppableService(name string, stoppable Stoppable) {
	c.RegisterShutdownService(name, NewStoppable(stoppable))
}

func (c *ControlFlowImpl) RegisterShutdownServiceWithName(service ShutdownWithName) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.services[service.Name()] = service
}

func (c *ControlFlowImpl) RegisterShutdownService(name string, service Shutdown) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.services[name] = service
}

func (c *ControlFlowImpl) Context() context.Context {
	return c.context
}

func (c *ControlFlowImpl) Cancel() context.CancelFunc {
	return c.cancel
}

func (c *ControlFlowImpl) WaitForInterrupt() {
	log := c.log.GetLogger(c.context)

	signal.NotifyContext(c.context)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)

	select {
	case <-c.context.Done():
		log.Info("context done, shutting down")
	case <-ch:
		log.Info("shutting down, canceling context")
		c.cancel()
	}
}

func (c *ControlFlowImpl) Shutdown() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if len(c.services) > 0 {
		log := c.log.GetLogger(c.context)

		log.Info("Shutting down...")

		wg := &sync.WaitGroup{}

		for serviceName, service := range c.services {
			wg.Go(func() {
				log.Infof("Shutting down service %s...", serviceName)

				if err := service.Stop(); err != nil {
					log.WithError(err).Errorf("Failed to shutdown service %s", serviceName)
				} else {
					log.Infof("Service %s shut down successfully", serviceName)
				}
			})
		}

		wg.Wait()

		log.Info("Shutdown complete")
	}
}
