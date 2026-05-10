package pool

import (
	"context"
	"sync"

	"github.com/pixality-inc/golang-core/errors"
	"github.com/pixality-inc/golang-core/logger"
)

var (
	ErrNotStarted     = errors.New("pool.not_started", "pool not started")
	ErrAlreadyStarted = errors.New("pool.already_started", "pool already started")
	ErrAlreadyStopped = errors.New("pool.already_stopped", "pool already stopped")
)

type PoolExecutor interface {
	Execute(ctx context.Context, functions ...TaskFunc) error
}

type PoolTaskExecutor interface {
	ExecuteTask(ctx context.Context, tasks ...Task) error
}

type Pool interface {
	PoolExecutor
	PoolTaskExecutor

	Start(ctx context.Context) error
	Stop() error
}

type Impl struct {
	log     logger.Loggable
	channel chan taskContext
	size    uint32
	started bool
	mutex   sync.Mutex
}

func New(name string, size uint32) Pool {
	return &Impl{
		log: logger.NewLoggableImplWithServiceAndFields("pool", logger.Fields{
			"name": name,
		}),
		channel: nil,
		size:    size,
		started: false,
		mutex:   sync.Mutex{},
	}
}

func (p *Impl) Execute(ctx context.Context, functions ...TaskFunc) error {
	for _, function := range functions {
		if err := p.ExecuteTask(ctx, NewTask(function)); err != nil {
			return err
		}
	}

	return nil
}

func (p *Impl) ExecuteTask(ctx context.Context, tasks ...Task) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.started {
		return ErrNotStarted
	}

	for _, task := range tasks {
		p.channel <- taskContext{
			ctx:  ctx,
			task: task,
		}
	}

	return nil
}

func (p *Impl) Start(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.started {
		return ErrAlreadyStarted
	}

	p.channel = make(chan taskContext)

	for i := range p.size {
		go p.startWorker(ctx, i+1)
	}

	p.started = true

	return nil
}

func (p *Impl) Stop() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.started {
		return ErrAlreadyStopped
	}

	close(p.channel)

	p.started = false

	return nil
}

func (p *Impl) startWorker(ctx context.Context, id uint32) {
	log := p.log.GetLogger(ctx).WithField("worker_id", id)

	for {
		select {
		case <-ctx.Done():
			log.WithError(ctx.Err()).Warn("Context stopped")

			return

		case taskCtx, ok := <-p.channel:
			if !ok {
				log.Warn("Channel closed")

				return
			}

			done := make(chan struct{})

			go func() {
				defer close(done)

				if fErr := taskCtx.task.Run(taskCtx.ctx); fErr != nil {
					log.WithError(fErr).Error("Task failed")
				}
			}()

			select {
			case <-ctx.Done():
				log.WithError(ctx.Err()).Warn("Context stopped")

				return

			case <-taskCtx.ctx.Done():
				log.WithError(taskCtx.ctx.Err()).Warn("Task context stopped")

			case <-done:
			}
		}
	}
}
