package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/pixality-inc/golang-core/logger"
	"github.com/pixality-inc/golang-core/timetrack"

	"github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4"
)

type (
	TransactionFunc             = func(runner QueryRunner) error
	TypedTransactionFunc[T any] = func(runner QueryRunner) (T, error)
)

func wrapQueryWithLogger(
	ctx context.Context,
	query Query,
	fn func(sqlQuery string, args []any, err error) error,
) error {
	log := logger.GetLogger(ctx)

	queryTimeTracker := timetrack.New(ctx)

	sqlQuery, args, queryErr := query.ToSql()

	baseLogger := func(isSuccess bool) logger.Logger {
		queryTimeTracker.Finish()

		return log.
			WithField("logger", "db").
			WithField("success", isSuccess).
			WithField("args_count", len(args)).
			WithField("execution_time", queryTimeTracker.Duration().Milliseconds())
	}

	logError := func(err error) {
		baseLogger(false).WithError(err).Error(sqlQuery)
	}

	logSuccess := func() {
		baseLogger(true).Debug(sqlQuery)
	}

	if queryErr != nil {
		queryErr = fmt.Errorf("query %+v failed: %w", query, queryErr)

		logError(queryErr)

		return queryErr
	}

	if err := fn(sqlQuery, args, queryErr); err != nil {
		err = fmt.Errorf("query %s failed with params %+v: %w", sqlQuery, args, err)

		logError(err)

		return err
	}

	logSuccess()

	return nil
}

func ExecuteQuery(
	ctx context.Context,
	queryRunner QueryRunner,
	query Query,
) error {
	queryExecutor, err := queryRunner.Executor()
	if err != nil {
		return err
	}

	return wrapQueryWithLogger(
		ctx,
		query,
		func(sqlQuery string, args []any, err error) error {
			if err != nil {
				return fmt.Errorf("sql build failed: %w", err)
			}

			_, err = queryExecutor.Exec(ctx, sqlQuery, args...)
			if err != nil {
				return fmt.Errorf("sql exec failed: %w", err)
			}

			return nil
		},
	)
}

func ExecuteQueryRows(ctx context.Context, queryRunner QueryRunner, query Query, dst any) error {
	queryExecutor, err := queryRunner.Executor()
	if err != nil {
		return err
	}

	return wrapQueryWithLogger(
		ctx,
		query,
		func(sqlQuery string, args []any, err error) error {
			if err != nil {
				return fmt.Errorf("sql build failed: %w", err)
			}

			rows, err := queryExecutor.Query(ctx, sqlQuery, args...)
			if err != nil {
				return fmt.Errorf("sql query failed: %w", err)
			}

			if err = pgxscan.ScanAll(dst, rows); !errors.Is(err, pgx.ErrNoRows) && err != nil {
				return fmt.Errorf("sql result scan failed: %w", err)
			}

			return nil
		},
	)
}

func ExecuteTransaction(
	ctx context.Context,
	name string,
	db Database,
	function TransactionFunc,
) error {
	_, err := ExecuteTypedTransaction[any](
		ctx,
		name,
		db,
		nil,
		func(runner QueryRunner) (any, error) {
			return nil, function(runner)
		},
	)

	return err
}

func ExecuteTypedTransaction[T any](
	ctx context.Context,
	name string,
	db Database,
	nullValue T,
	function TypedTransactionFunc[T],
) (T, error) {
	transactionTimeTracker := timetrack.New(ctx)

	result := nullValue

	err := db.BeginTxFunc(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		var err error

		result, err = function(NewQueryExecutorImpl(db.Name()+":tx", tx, db.GetCircuitBreaker()))

		return err
	})

	transactionTimeTracker.Finish()

	baseLogger := func(isSuccess bool) logger.Logger {
		return logger.GetLogger(ctx).
			WithField("transaction", name).
			WithField("logger", "db_transaction").
			WithField("success", isSuccess).
			WithField("execution_time", transactionTimeTracker.Duration().Milliseconds())
	}

	if err != nil {
		baseLogger(false).WithError(err).Errorf("transaction failed")
	} else {
		baseLogger(true).Debugf("transaction executed")
	}

	return result, err
}

func BuildSimpleInsertQuery[T any](
	query squirrel.InsertBuilder,
	columns []GetterInsertColumn[T],
	row T,
) (squirrel.InsertBuilder, error) {
	values := make([]any, 0, len(columns))

	for _, column := range columns {
		query = query.Columns(column.Name)

		values = append(values, column.Getter(row))
	}

	query = query.Values(values...)

	return query, nil
}

func BuildBulkInsertQuery[T any](
	query squirrel.InsertBuilder,
	columns []GetterInsertColumn[T],
	rows []T,
) (squirrel.InsertBuilder, error) {
	for _, column := range columns {
		query = query.Columns(column.Name)
	}

	for _, row := range rows {
		var values []any // nolint:prealloc

		for _, column := range columns {
			values = append(values, column.Getter(row))
		}

		query = query.Values(values...)
	}

	return query, nil
}
