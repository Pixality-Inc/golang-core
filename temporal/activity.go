package temporal

import (
	"context"
	"time"

	"github.com/pixality-inc/golang-core/logger"
)

//go:generate mockgen -destination mocks/activity_gen.go -source activity.go
type Activity interface {
	Name() ActivityName
	Queue() QueueName
	Timeout() time.Duration
	MaxAttempts() int
	RetryInitialInterval() time.Duration
	RetryBackoffCoefficient() float64
	RetryMaximumInterval() time.Duration
}

type ActivityTypedWrapper[IN any, OUT any] struct {
	activity Activity
	runner   ActivityTypedRunner[IN, OUT]
}

func NewActivityTypedWrapper[IN any, OUT any](a Activity, runner ActivityTypedRunner[IN, OUT]) *ActivityTypedWrapper[IN, OUT] {
	return &ActivityTypedWrapper[IN, OUT]{
		activity: a,
		runner:   runner,
	}
}

type ActivityImpl struct {
	log    logger.Loggable
	worker Worker
	config ActivityConfig
}

func NewActivityImpl(
	worker Worker,
	config ActivityConfig,
) *ActivityImpl {
	return &ActivityImpl{
		log: logger.NewLoggableImplWithServiceAndFields(
			"temporal_activity",
			logger.Fields{
				"name": config.Name,
			},
		),
		worker: worker,
		config: config,
	}
}

func (a *ActivityImpl) Name() ActivityName {
	return a.config.Name
}

func (a *ActivityImpl) Queue() QueueName {
	return a.config.Queue
}

func (a *ActivityImpl) Timeout() time.Duration {
	return a.config.Timeout
}

func (a *ActivityImpl) MaxAttempts() int {
	return a.config.MaxAttempts
}

func (a *ActivityImpl) RetryInitialInterval() time.Duration {
	return a.config.RetryInitialInterval
}

func (a *ActivityImpl) RetryBackoffCoefficient() float64 {
	return a.config.RetryBackoffCoefficient
}

func (a *ActivityImpl) RetryMaximumInterval() time.Duration {
	return a.config.RetryMaximumInterval
}

func (a *ActivityImpl) GetLogger(ctx context.Context) logger.Logger {
	return a.log.GetLogger(ctx)
}

func (a *ActivityImpl) GetLoggerWithoutContext() logger.Logger {
	return a.log.GetLoggerWithoutContext()
}
