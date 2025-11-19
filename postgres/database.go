package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"

	"github.com/pixality-inc/golang-core/circuit_breaker"
	"github.com/pixality-inc/golang-core/logger"

	"github.com/jackc/pgx/v4"

	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	_defaultMaxPoolSize = 1
)

var errInvalidMaxPoolSize = errors.New("invalid maxPoolSize")

type Database interface {
	Close()
	Name() string
	Executor() (QueryExecutor, error)
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
	BeginTxFunc(ctx context.Context, opts pgx.TxOptions, f func(pgx.Tx) error) error
	IsConnected() bool
	GetCircuitBreaker() circuit_breaker.CircuitBreaker
}

type DatabaseImpl struct {
	log               logger.Loggable
	ctx               context.Context //nolint:containedctx // needed for lazy initialization of db connections
	name              string
	url               string
	maxPoolSize       int
	poolConfig        *pgxpool.Config
	pool              *pgxpool.Pool
	poolQueryExecutor QueryExecutor
	connected         bool
	mutex             sync.Mutex
	circuitBreaker    circuit_breaker.CircuitBreaker
}

func New(
	ctx context.Context,
	name string,
	url string,
	opts ...Option,
) (Database, error) {
	database := &DatabaseImpl{
		log: logger.NewLoggableImplWithServiceAndFields(
			"db",
			logger.Fields{
				"name": name,
			},
		),
		ctx:            ctx,
		name:           name,
		url:            url,
		maxPoolSize:    _defaultMaxPoolSize,
		poolConfig:     nil,
		pool:           nil,
		connected:      false,
		mutex:          sync.Mutex{},
		circuitBreaker: nil,
	}

	for _, opt := range opts {
		opt(database)
	}

	poolConfig, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}

	if database.maxPoolSize > math.MaxInt32 || database.maxPoolSize < 0 {
		return nil, fmt.Errorf("%w: %d (must be between 0 and %d)", errInvalidMaxPoolSize, database.maxPoolSize, math.MaxInt32)
	}

	poolConfig.MaxConns = int32(database.maxPoolSize)

	database.poolConfig = poolConfig

	return database, nil
}

func (d *DatabaseImpl) Name() string {
	return d.name
}

func (d *DatabaseImpl) Executor() (QueryExecutor, error) {
	if err := d.ensureConnected(); err != nil {
		return nil, err
	}

	return d.poolQueryExecutor, nil
}

func (d *DatabaseImpl) BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error) {
	if err := d.ensureConnected(); err != nil {
		return nil, err
	}

	return circuit_breaker.ExecuteWithResult(
		d.circuitBreaker,
		func() (pgx.Tx, error) {
			return d.pool.BeginTx(ctx, opts)
		},
		nil,
	)
}

func (d *DatabaseImpl) BeginTxFunc(ctx context.Context, opts pgx.TxOptions, txFunc func(pgx.Tx) error) error {
	if err := d.ensureConnected(); err != nil {
		return err
	}

	return circuit_breaker.Execute(d.circuitBreaker, func() error {
		return d.pool.BeginTxFunc(ctx, opts, txFunc)
	})
}

func (d *DatabaseImpl) Close() {
	if d.pool != nil {
		d.pool.Close()
	}
}

func (d *DatabaseImpl) IsConnected() bool {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	return d.connected
}

func (d *DatabaseImpl) GetCircuitBreaker() circuit_breaker.CircuitBreaker {
	return d.circuitBreaker
}

func (d *DatabaseImpl) ensureConnected() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.connected {
		return nil
	}

	connectFunc := func() error {
		thePool, err := pgxpool.ConnectConfig(d.ctx, d.poolConfig)
		if err != nil {
			return err
		}

		if err = thePool.Ping(d.ctx); err != nil {
			return err
		}

		d.pool = thePool
		d.poolQueryExecutor = NewQueryExecutorImpl(d.name, d.pool, d.circuitBreaker)
		d.connected = true

		return nil
	}

	return circuit_breaker.Execute(d.circuitBreaker, connectFunc)
}
