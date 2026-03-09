package kafka

import (
	"context"
	"fmt"
	"sync"

	"github.com/pixality-inc/golang-core/circuit_breaker"
	"github.com/pixality-inc/golang-core/logger"
	"github.com/pixality-inc/golang-core/retry"

	"github.com/twmb/franz-go/pkg/kgo"
)

type Producer[T any] interface {
	Produce(ctx context.Context, value T, opts ...ProduceOption) error
	ProduceBatch(ctx context.Context, values []T, opts ...ProduceOption) error
	IsConnected() bool
	Ping(ctx context.Context) error
	Stop() error
}

// BatchProduceError is returned by ProduceBatch when some records in the batch fail.
// Errors slice has the same length as the input values: nil entries indicate success.
type BatchProduceError struct {
	Errors []error
}

func (e *BatchProduceError) Error() string {
	var count int

	for _, err := range e.Errors {
		if err != nil {
			count++
		}
	}

	return fmt.Sprintf("batch produce: %d of %d records failed", count, len(e.Errors))
}

func (e *BatchProduceError) Unwrap() []error {
	errs := make([]error, 0, len(e.Errors))
	for _, err := range e.Errors {
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

type producerImpl[T any] struct {
	log            logger.Loggable
	config         Config
	protocol       Protocol[T]
	client         *kgo.Client
	mutex          sync.RWMutex
	circuitBreaker circuit_breaker.CircuitBreaker
	retryPolicy    retry.Policy
}

func NewProducer[T any](config Config, protocol Protocol[T], opts ...Option) (Producer[T], error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	options := applyOptions(opts...)

	var cb circuit_breaker.CircuitBreaker
	if options.circuitBreaker != nil {
		cb = options.circuitBreaker
	} else if config.CircuitBreaker() != nil {
		cb = NewCircuitBreaker(config.CircuitBreaker(), nil)
	}

	retryPolicy := config.RetryPolicy()
	if options.retryPolicy != nil {
		retryPolicy = options.retryPolicy
	}

	return &producerImpl[T]{
		log: logger.NewLoggableImplWithServiceAndFields("kafka_producer", logger.Fields{
			"topic": config.Topic(),
		}),
		config:         config,
		protocol:       protocol,
		circuitBreaker: cb,
		retryPolicy:    retryPolicy,
	}, nil
}

func (p *producerImpl[T]) ensureConnected(ctx context.Context) (*kgo.Client, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.client != nil {
		return p.client, nil
	}

	connectFunc := func() error {
		kgoOpts, err := buildKgoOpts(p.config)
		if err != nil {
			return err
		}

		kgoOpts = append(kgoOpts, kgo.DefaultProduceTopic(p.config.Topic()))

		client, err := kgo.NewClient(kgoOpts...)
		if err != nil {
			return fmt.Errorf("failed to create kafka producer client: %w", err)
		}

		pingCtx, cancel := context.WithTimeout(ctx, p.config.ConnectTimeout())
		defer cancel()

		if err = client.Ping(pingCtx); err != nil {
			client.Close()

			return fmt.Errorf("failed to ping kafka broker: %w", err)
		}

		p.client = client

		return nil
	}

	if err := circuit_breaker.Execute(p.circuitBreaker, connectFunc); err != nil {
		return nil, err
	}

	return p.client, nil
}

func (p *producerImpl[T]) Produce(ctx context.Context, value T, opts ...ProduceOption) error {
	client, err := p.ensureConnected(ctx)
	if err != nil {
		return err
	}

	data, err := p.protocol.Encode(ctx, value)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	record := buildRecord(p.config.Topic(), data, applyProduceOptions(opts...))

	_, err = retry.Do(ctx, p.retryPolicy, p.log, func() (struct{}, error) {
		produceErr := circuit_breaker.Execute(p.circuitBreaker, func() error {
			return client.ProduceSync(ctx, record).FirstErr()
		})

		return struct{}{}, produceErr
	})

	return err
}

func (p *producerImpl[T]) ProduceBatch(ctx context.Context, values []T, opts ...ProduceOption) error {
	client, err := p.ensureConnected(ctx)
	if err != nil {
		return err
	}

	cfg := applyProduceOptions(opts...)
	records := make([]*kgo.Record, 0, len(values))

	for _, value := range values {
		data, err := p.protocol.Encode(ctx, value)
		if err != nil {
			return fmt.Errorf("failed to encode message: %w", err)
		}

		records = append(records, buildRecord(p.config.Topic(), data, cfg))
	}

	_, err = retry.Do(ctx, p.retryPolicy, p.log, func() (struct{}, error) {
		produceErr := circuit_breaker.Execute(p.circuitBreaker, func() error {
			results := client.ProduceSync(ctx, records...)

			var hasError bool

			errs := make([]error, len(results))

			for i, r := range results {
				if r.Err != nil {
					errs[i] = r.Err
					hasError = true
				}
			}

			if hasError {
				return &BatchProduceError{Errors: errs}
			}

			return nil
		})

		return struct{}{}, produceErr
	})

	return err
}

func (p *producerImpl[T]) IsConnected() bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.client != nil
}

func (p *producerImpl[T]) Ping(ctx context.Context) error {
	p.mutex.RLock()
	client := p.client
	p.mutex.RUnlock()

	if client == nil {
		return ErrNotConnected
	}

	return client.Ping(ctx)
}

func (p *producerImpl[T]) Stop() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.client != nil {
		p.client.Close()
		p.client = nil
	}

	return nil
}
