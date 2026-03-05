package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/pixality-inc/golang-core/logger"
	"github.com/pixality-inc/golang-core/timetrack"
	"github.com/pixality-inc/golang-core/util"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/pixality-inc/squirrel"
)

type (
	TransactionFunc             = func(runner QueryRunner) error
	TypedTransactionFunc[T any] = func(runner QueryRunner) (T, error)
)

func wrapQueryWithLogger(
	ctx context.Context,
	query Query,
	fn func(sqlQuery string, args []any) (QueryResult, error),
) (QueryResult, error) {
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

		return nil, queryErr
	}

	result, err := fn(sqlQuery, args)
	if err != nil {
		err = fmt.Errorf("query %s failed with params %+v: %w", sqlQuery, args, err)

		logError(err)

		return nil, err
	}

	logSuccess()

	return result, nil
}

func ExecuteQuery(
	ctx context.Context,
	queryRunner QueryRunner,
	query Query,
) (QueryResult, error) {
	queryExecutor, err := queryRunner.Executor()
	if err != nil {
		return nil, err
	}

	return wrapQueryWithLogger(
		ctx,
		query,
		func(sqlQuery string, args []any) (QueryResult, error) {
			result, err := queryExecutor.Exec(ctx, sqlQuery, args...)
			if err != nil {
				return nil, fmt.Errorf("sql exec failed: %w", err)
			}

			return NewQueryResult(result.RowsAffected()), nil
		},
	)
}

func ExecuteQueryRows(ctx context.Context, queryRunner QueryRunner, query Query, dst any) (QueryResult, error) {
	queryExecutor, err := queryRunner.Executor()
	if err != nil {
		return nil, err
	}

	return wrapQueryWithLogger(
		ctx,
		query,
		func(sqlQuery string, args []any) (QueryResult, error) {
			rows, err := queryExecutor.Query(ctx, sqlQuery, args...)
			if err != nil {
				return nil, fmt.Errorf("sql query failed: %w", err)
			}

			if err = pgxscan.ScanAll(dst, rows); !errors.Is(err, pgx.ErrNoRows) && err != nil {
				return nil, fmt.Errorf("sql result scan failed: %w", err)
			}

			return NewEmptyQueryResult(), nil
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
		values := make([]any, 0, len(columns))

		for _, column := range columns {
			values = append(values, column.Getter(row))
		}

		query = query.Values(values...)
	}

	return query, nil
}

func FetchRowsSimple[R any](
	ctx context.Context,
	queryRunner QueryRunner,
	query Query,
) ([]R, error) {
	var rows []R

	if _, err := ExecuteQueryRows(ctx, queryRunner, query, &rows); err != nil {
		return nil, err
	}

	return rows, nil
}

func FetchRows[R any, M any](
	ctx context.Context,
	queryRunner QueryRunner,
	query Query,
	convert func(R) M,
) ([]M, error) {
	rows, err := FetchRowsSimple[R](ctx, queryRunner, query)
	if err != nil {
		return nil, err
	}

	return util.MapSimple(rows, convert), nil
}

func FetchRowSimple[R any](
	ctx context.Context,
	queryRunner QueryRunner,
	query Query,
	defaultValue R,
) (R, error) {
	var rows []R

	if _, err := ExecuteQueryRows(ctx, queryRunner, query, &rows); err != nil {
		return defaultValue, err
	}

	if len(rows) > 0 {
		return rows[0], nil
	}

	return defaultValue, ErrNoRows
}

func FetchRow[R any, M any](
	ctx context.Context,
	queryRunner QueryRunner,
	query Query,
	convert func(R) M,
	defaultValue M,
) (M, error) {
	var rows []R

	if _, err := ExecuteQueryRows(ctx, queryRunner, query, &rows); err != nil {
		return defaultValue, err
	}

	if len(rows) > 0 {
		return convert(rows[0]), nil
	}

	return defaultValue, ErrNoRows
}
