package pool

import (
	"context"

	"github.com/pixality-inc/golang-core/logger"
)

type DefaultImpl struct{}

var Default = NewDefault()

func NewDefault() Pool {
	return &DefaultImpl{}
}

func (p *DefaultImpl) Start(_ context.Context) error {
	return nil
}

func (p *DefaultImpl) Stop() error {
	return nil
}

func (p *DefaultImpl) Execute(ctx context.Context, functions ...TaskFunc) error {
	for _, function := range functions {
		if err := p.ExecuteTask(ctx, NewTask(function)); err != nil {
			return err
		}
	}

	return nil
}

func (p *DefaultImpl) ExecuteTask(ctx context.Context, tasks ...Task) error {
	log := logger.GetLogger(ctx)

	for _, task := range tasks {
		go func() {
			if fErr := task.Run(ctx); fErr != nil {
				log.WithError(fErr).Error("Task failed")
			}
		}()
	}

	return nil
}
