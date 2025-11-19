package postgres

import (
	"context"
	"testing"

	"github.com/pixality-inc/golang-core/circuit_breaker"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/require"
)

type fakeCircuitBreakerUnexpectedType struct{}

func (f *fakeCircuitBreakerUnexpectedType) Execute(fn func() error) error {
	return nil
}

func (f *fakeCircuitBreakerUnexpectedType) ExecuteWithResult(fn func() (any, error)) (any, error) {
	// Return deliberately wrong type to trigger type assertion error.
	return "not-a-command-tag", nil
}

type fakeQueryExecutor struct{}

func (f *fakeQueryExecutor) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (f *fakeQueryExecutor) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, nil
}

func TestQueryExecutorImpl_ExecUnexpectedTypeError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cb := &fakeCircuitBreakerUnexpectedType{}
	executor := &fakeQueryExecutor{}

	qe := NewQueryExecutorImpl("fake", executor, cb)

	_, err := qe.Exec(ctx, "SELECT 1")
	require.Error(t, err)
	require.ErrorIs(t, err, circuit_breaker.ErrUnexpectedType)
}
