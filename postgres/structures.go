package postgres

import (
	"context"

	"github.com/pixality-inc/golang-core/circuit_breaker"
	"github.com/pixality-inc/golang-core/logger"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

type Query interface {
	ToSql() (string, []any, error)
}

type QueryRunner interface {
	Executor() (QueryExecutor, error)
}

type QueryExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type QueryExecutorImpl struct {
	log            logger.Loggable
	name           string
	executor       QueryExecutor
	circuitBreaker circuit_breaker.CircuitBreaker
}

func NewQueryExecutorImpl(
	name string,
	executor QueryExecutor,
	cb circuit_breaker.CircuitBreaker,
) *QueryExecutorImpl {
	return &QueryExecutorImpl{
		log: logger.NewLoggableImplWithServiceAndFields(
			"query_executor",
			logger.Fields{
				"name": name,
			},
		),
		name:           name,
		executor:       executor,
		circuitBreaker: cb,
	}
}

func (q *QueryExecutorImpl) Executor() (QueryExecutor, error) {
	return q.executor, nil
}

func (q *QueryExecutorImpl) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return circuit_breaker.ExecuteWithResult(
		q.circuitBreaker,
		func() (pgconn.CommandTag, error) {
			return q.executor.Exec(ctx, sql, arguments...)
		},
		pgconn.CommandTag{},
	)
}

func (q *QueryExecutorImpl) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return circuit_breaker.ExecuteWithResult(
		q.circuitBreaker,
		func() (pgx.Rows, error) {
			return q.executor.Query(ctx, sql, args...)
		},
		nil,
	)
}

type InsertColumn struct {
	Name  string
	Value any
}

func NewInsertColumn(name string, value any) InsertColumn {
	return InsertColumn{
		Name:  name,
		Value: value,
	}
}

type GetterInsertColumn[T any] struct {
	Name   string
	Getter func(obj T) any
}

func NewGetterInsertColumn[T any](name string, getter func(obj T) any) GetterInsertColumn[T] {
	return GetterInsertColumn[T]{
		Name:   name,
		Getter: getter,
	}
}
