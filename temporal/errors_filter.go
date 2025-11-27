package temporal

import (
	"context"
	"errors"

	cb "github.com/pixality-inc/golang-core/circuit_breaker"

	"go.temporal.io/api/serviceerror"
)

// ShouldIgnoreErrorForCircuitBreaker determines if a temporal error should be ignored
// by the circuit breaker. returns true for errors that are expected workflow/application logic results
// rather than infrastructure/availability issues.
func ShouldIgnoreErrorForCircuitBreaker(err error) bool {
	if err == nil {
		return false
	}

	// context cancellation from application level - not an infrastructure issue
	if errors.Is(err, context.Canceled) {
		return true
	}

	// workflow/application logic errors - these should not trigger circuit breaker
	// these are client-side or business logic errors

	// WorkflowExecutionAlreadyStarted - workflow with same ID already running, business logic issue
	var alreadyStartedErr *serviceerror.WorkflowExecutionAlreadyStarted
	if errors.As(err, &alreadyStartedErr) {
		return true
	}

	// InvalidArgument - client sent invalid request, business logic issue
	var invalidArgErr *serviceerror.InvalidArgument
	if errors.As(err, &invalidArgErr) {
		return true
	}

	// NotFound - workflow/namespace not found, business logic issue
	var notFoundErr *serviceerror.NotFound
	if errors.As(err, &notFoundErr) {
		return true
	}

	// AlreadyExists - namespace/schedule already exists, business logic issue
	var alreadyExistsErr *serviceerror.AlreadyExists
	if errors.As(err, &alreadyExistsErr) {
		return true
	}

	// PermissionDenied - authorization issue, not infrastructure problem
	var permissionDeniedErr *serviceerror.PermissionDenied
	if errors.As(err, &permissionDeniedErr) {
		return true
	}

	// FailedPrecondition - workflow state precondition failed, business logic issue
	var failedPreconditionErr *serviceerror.FailedPrecondition
	if errors.As(err, &failedPreconditionErr) {
		return true
	}

	// Canceled - operation was canceled by client, not infrastructure issue
	var canceledErr *serviceerror.Canceled
	if errors.As(err, &canceledErr) {
		return true
	}

	// QueryFailed - query execution failed due to workflow state, business logic
	var queryFailedErr *serviceerror.QueryFailed
	if errors.As(err, &queryFailedErr) {
		return true
	}

	// NamespaceNotFound - namespace doesn't exist, configuration issue
	var namespaceNotFoundErr *serviceerror.NamespaceNotFound
	if errors.As(err, &namespaceNotFoundErr) {
		return true
	}

	// NamespaceInvalidState - namespace in invalid state, configuration issue
	var namespaceInvalidStateErr *serviceerror.NamespaceInvalidState
	if errors.As(err, &namespaceInvalidStateErr) {
		return true
	}

	// infrastructure/availability errors - these SHOULD trigger circuit breaker

	// Unavailable - temporal server unavailable, infrastructure issue
	var unavailableErr *serviceerror.Unavailable
	if errors.As(err, &unavailableErr) {
		return false
	}

	// DeadlineExceeded - request deadline exceeded, infrastructure issue
	var deadlineExceededErr *serviceerror.DeadlineExceeded
	if errors.As(err, &deadlineExceededErr) {
		return false
	}

	// ResourceExhausted - server overloaded, infrastructure issue
	var resourceExhaustedErr *serviceerror.ResourceExhausted
	if errors.As(err, &resourceExhaustedErr) {
		return false
	}

	// Internal - internal server error, infrastructure issue
	var internalErr *serviceerror.Internal
	if errors.As(err, &internalErr) {
		return false
	}

	// DataLoss - unrecoverable data loss, infrastructure issue
	var dataLossErr *serviceerror.DataLoss
	if errors.As(err, &dataLossErr) {
		return false
	}

	// all other errors are considered infrastructure issues
	return false
}

// NewCircuitBreaker creates a circuit breaker configured with Temporal-specific error filtering.
func NewCircuitBreaker(config cb.Config, shouldIgnoreError func(err error) bool) cb.CircuitBreaker {
	if shouldIgnoreError == nil {
		shouldIgnoreError = ShouldIgnoreErrorForCircuitBreaker
	}

	return cb.New(config, shouldIgnoreError)
}
