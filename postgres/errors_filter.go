package postgres

import (
	"context"
	"database/sql"
	"errors"

	cb "github.com/pixality-inc/golang-core/circuit_breaker"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

// ShouldIgnoreErrorForCircuitBreaker determines if a postgres error should be ignored
// by the circuit breaker. Returns true for errors that are expected business logic results
// rather than infrastructure/availability issues.
func ShouldIgnoreErrorForCircuitBreaker(err error) bool {
	// No rows found - expected business logic result
	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
		return true
	}

	// Context cancellation from application level - not a DB issue
	if errors.Is(err, context.Canceled) {
		return true
	}

	// Check for postgres-specific error codes
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return shouldIgnorePgError(pgErr)
	}

	return false
}

// NewCircuitBreaker creates a circuit breaker configured with postgres-specific error filtering.
func NewCircuitBreaker(config cb.Config, shouldIgnoreError func(err error) bool) cb.CircuitBreaker {
	if shouldIgnoreError == nil {
		shouldIgnoreError = ShouldIgnoreErrorForCircuitBreaker
	}

	return cb.New(config, shouldIgnoreError)
}

// shouldIgnorePgError determines if a postgres-specific error should be ignored.
func shouldIgnorePgError(pgErr *pgconn.PgError) bool {
	if len(pgErr.Code) < 2 {
		return false
	}

	// starting with 23 - integrity constraint violation, these are not database availability issues
	// 23000: integrity_constraint_violation
	// 23001: restrict_violation
	// 23502: not_null_violation
	// 23503: foreign_key_violation
	// 23505: unique_violation
	// 23514: check_violation
	// 23P01: exclusion_violation
	if pgErr.Code[:2] == "23" {
		return true
	}

	// starting with 42 - syntax error or access rule violation. these are not database availability issues
	// 42601: syntax_error
	// 42501: insufficient_privilege
	// 42846: cannot_coerce
	// 42803: grouping_error
	if pgErr.Code[:2] == "42" {
		return true
	}

	// starting with 40 - transaction related issues, obv not database availability issues
	// 40001: serialization_failure
	// 40P01: deadlock_detected
	if pgErr.Code == "40001" || pgErr.Code == "40P01" {
		return true
	}

	// starting with 22 - data exception, obv not database availability issues
	// 22000: data_exception
	// 22001: string_data_right_truncation
	// 22008: datetime_field_overflow
	// 22012: division_by_zero
	// 22P02: invalid_text_representation
	if pgErr.Code[:2] == "22" {
		return true
	}

	return false
}
