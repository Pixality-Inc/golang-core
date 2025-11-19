package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/require"
)

var (
	errSomeGeneric = errors.New("some generic error")
	errWrapped     = errors.New("wrapped: some error")
)

func TestShouldIgnoreErrorForCircuitBreaker_ErrNoRows(t *testing.T) {
	t.Parallel()

	// Test pgx.ErrNoRows
	shouldIgnore := ShouldIgnoreErrorForCircuitBreaker(pgx.ErrNoRows)
	require.True(t, shouldIgnore, "pgx.ErrNoRows should be ignored")

	// Test sql.ErrNoRows
	shouldIgnore = ShouldIgnoreErrorForCircuitBreaker(sql.ErrNoRows)
	require.True(t, shouldIgnore, "sql.ErrNoRows should be ignored")
}

func TestShouldIgnoreErrorForCircuitBreaker_ContextCanceled(t *testing.T) {
	t.Parallel()

	shouldIgnore := ShouldIgnoreErrorForCircuitBreaker(context.Canceled)
	require.True(t, shouldIgnore, "context.Canceled should be ignored")
}

func TestShouldIgnoreErrorForCircuitBreaker_ConstraintViolations(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		code string
		desc string
	}{
		{"unique_violation", "23505", "unique constraint violation"},
		{"foreign_key_violation", "23503", "foreign key constraint violation"},
		{"not_null_violation", "23502", "not null constraint violation"},
		{"check_violation", "23514", "check constraint violation"},
		{"exclusion_violation", "23P01", "exclusion constraint violation"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			pgErr := &pgconn.PgError{
				Code:    testCase.code,
				Message: testCase.desc,
			}

			shouldIgnore := ShouldIgnoreErrorForCircuitBreaker(pgErr)
			require.True(t, shouldIgnore, "%s should be ignored", testCase.desc)
		})
	}
}

func TestShouldIgnoreErrorForCircuitBreaker_SyntaxErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		code string
		desc string
	}{
		{"syntax_error", "42601", "syntax error"},
		{"insufficient_privilege", "42501", "insufficient privilege"},
		{"cannot_coerce", "42846", "cannot coerce"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			pgErr := &pgconn.PgError{
				Code:    testCase.code,
				Message: testCase.desc,
			}

			shouldIgnore := ShouldIgnoreErrorForCircuitBreaker(pgErr)
			require.True(t, shouldIgnore, "%s should be ignored", testCase.desc)
		})
	}
}

func TestShouldIgnoreErrorForCircuitBreaker_DataExceptions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		code string
		desc string
	}{
		{"division_by_zero", "22012", "division by zero"},
		{"invalid_text_representation", "22P02", "invalid input syntax"},
		{"string_data_right_truncation", "22001", "value too long"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			pgErr := &pgconn.PgError{
				Code:    testCase.code,
				Message: testCase.desc,
			}

			shouldIgnore := ShouldIgnoreErrorForCircuitBreaker(pgErr)
			require.True(t, shouldIgnore, "%s should be ignored", testCase.desc)
		})
	}
}

func TestShouldIgnoreErrorForCircuitBreaker_TransactionRollback(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		code string
		desc string
	}{
		{"serialization_failure", "40001", "could not serialize access"},
		{"deadlock_detected", "40P01", "deadlock detected"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			pgErr := &pgconn.PgError{
				Code:    testCase.code,
				Message: testCase.desc,
			}

			shouldIgnore := ShouldIgnoreErrorForCircuitBreaker(pgErr)
			require.True(t, shouldIgnore, "%s should be ignored", testCase.desc)
		})
	}
}

func TestShouldIgnoreErrorForCircuitBreaker_ConnectionErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		code string
		desc string
	}{
		{"connection_exception", "08000", "connection exception"},
		{"connection_failure", "08006", "connection failure"},
		{"connection_does_not_exist", "08003", "connection does not exist"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			pgErr := &pgconn.PgError{
				Code:    testCase.code,
				Message: testCase.desc,
			}

			shouldIgnore := ShouldIgnoreErrorForCircuitBreaker(pgErr)
			require.False(t, shouldIgnore, "%s SHOULD trigger circuit breaker", testCase.desc)
		})
	}
}

func TestShouldIgnoreErrorForCircuitBreaker_GenericErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		err  error
		desc string
	}{
		{
			"generic_error",
			errSomeGeneric,
			"generic error",
		},
		{
			"wrapped_generic_error",
			errWrapped,
			"wrapped generic error",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			shouldIgnore := ShouldIgnoreErrorForCircuitBreaker(testCase.err)
			require.False(t, shouldIgnore, "%s SHOULD trigger circuit breaker", testCase.desc)
		})
	}
}

func TestShouldIgnoreErrorForCircuitBreaker_NilError(t *testing.T) {
	t.Parallel()

	shouldIgnore := ShouldIgnoreErrorForCircuitBreaker(nil)
	require.False(t, shouldIgnore, "nil error should return false")
}

func TestShouldIgnoreErrorForCircuitBreaker_ShortErrorCode(t *testing.T) {
	t.Parallel()

	// Test PgError with too short code
	pgErr := &pgconn.PgError{
		Code:    "1",
		Message: "short code",
	}

	shouldIgnore := ShouldIgnoreErrorForCircuitBreaker(pgErr)
	require.False(t, shouldIgnore, "error with short code should not be ignored")
}
