package circuit_breaker

import (
	"github.com/pixality-inc/golang-core/logger"

	"github.com/sony/gobreaker/v2"
)

type gobreakerImpl struct {
	log logger.Loggable
	cb  *gobreaker.CircuitBreaker[any]
}

func newGobreakerImpl(config Config, shouldIgnoreError func(err error) bool) CircuitBreaker {
	log := logger.NewLoggableImplWithServiceAndFields(
		"circuit_breaker",
		logger.Fields{
			"name": config.Name,
		},
	)

	settings := gobreaker.Settings{
		Name:         config.Name(),
		MaxRequests:  config.MaxRequests(),
		Interval:     config.Interval(),
		Timeout:      config.Timeout(),
		BucketPeriod: config.BucketPeriod(),
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureThreshold := config.ConsecutiveFailures()
			if failureThreshold == 0 {
				failureThreshold = DefaultConsecutiveFailures
			}

			return counts.ConsecutiveFailures >= failureThreshold
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			log.GetLoggerWithoutContext().
				WithField("from", from.String()).
				WithField("to", to.String()).
				Warnf("circuit breaker state changed: %s -> %s", from.String(), to.String())
		},
		IsSuccessful: func(err error) bool {
			if err == nil {
				return true
			}

			// Use custom error filter if provided
			if shouldIgnoreError != nil && shouldIgnoreError(err) {
				return true
			}

			return false
		},
	}

	cb := gobreaker.NewCircuitBreaker[any](settings)

	return &gobreakerImpl{
		log: log,
		cb:  cb,
	}
}

func (g *gobreakerImpl) Execute(fn func() error) error {
	_, err := g.cb.Execute(func() (any, error) {
		return struct{}{}, fn()
	})

	return err
}

func (g *gobreakerImpl) ExecuteWithResult(fn func() (any, error)) (any, error) {
	return g.cb.Execute(fn)
}
