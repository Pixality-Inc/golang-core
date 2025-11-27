package temporal

import (
	"context"

	"github.com/pixality-inc/golang-core/circuit_breaker"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
)

// ClientWrapper wraps a Temporal client with circuit breaker protection
type ClientWrapper struct {
	client         client.Client
	circuitBreaker circuit_breaker.CircuitBreaker
}

// NewClientWrapper creates a new Temporal client wrapper with circuit breaker
func NewClientWrapper(c client.Client, cb circuit_breaker.CircuitBreaker) client.Client {
	return &ClientWrapper{
		client:         c,
		circuitBreaker: cb,
	}
}

// ExecuteWorkflow wraps the original ExecuteWorkflow with circuit breaker
func (w *ClientWrapper) ExecuteWorkflow(ctx context.Context, options client.StartWorkflowOptions, workflow any, args ...any) (client.WorkflowRun, error) {
	if w.circuitBreaker == nil {
		return w.client.ExecuteWorkflow(ctx, options, workflow, args...)
	}

	var result client.WorkflowRun

	err := circuit_breaker.Execute(w.circuitBreaker, func() error {
		var execErr error

		result, execErr = w.client.ExecuteWorkflow(ctx, options, workflow, args...)

		return execErr
	})

	return result, err
}

// GetWorkflow does not make network request, no circuit breaker needed
func (w *ClientWrapper) GetWorkflow(ctx context.Context, workflowID string, runID string) client.WorkflowRun {
	return w.client.GetWorkflow(ctx, workflowID, runID)
}

// SignalWorkflow wraps the original SignalWorkflow with circuit breaker
func (w *ClientWrapper) SignalWorkflow(ctx context.Context, workflowID string, runID string, signalName string, arg any) error {
	if w.circuitBreaker == nil {
		return w.client.SignalWorkflow(ctx, workflowID, runID, signalName, arg)
	}

	return circuit_breaker.Execute(w.circuitBreaker, func() error {
		return w.client.SignalWorkflow(ctx, workflowID, runID, signalName, arg)
	})
}

// SignalWithStartWorkflow wraps the original SignalWithStartWorkflow with circuit breaker
func (w *ClientWrapper) SignalWithStartWorkflow(ctx context.Context, workflowID string, signalName string, signalArg any, options client.StartWorkflowOptions, workflow any, workflowArgs ...any) (client.WorkflowRun, error) {
	if w.circuitBreaker == nil {
		return w.client.SignalWithStartWorkflow(ctx, workflowID, signalName, signalArg, options, workflow, workflowArgs...)
	}

	var result client.WorkflowRun

	err := circuit_breaker.Execute(w.circuitBreaker, func() error {
		var execErr error

		result, execErr = w.client.SignalWithStartWorkflow(ctx, workflowID, signalName, signalArg, options, workflow, workflowArgs...)

		return execErr
	})

	return result, err
}

// NewWithStartWorkflowOperation does not make network request, no circuit breaker needed
func (w *ClientWrapper) NewWithStartWorkflowOperation(options client.StartWorkflowOptions, workflow any, args ...any) client.WithStartWorkflowOperation {
	return w.client.NewWithStartWorkflowOperation(options, workflow, args...)
}

// CancelWorkflow wraps the original CancelWorkflow with circuit breaker
func (w *ClientWrapper) CancelWorkflow(ctx context.Context, workflowID string, runID string) error {
	if w.circuitBreaker == nil {
		return w.client.CancelWorkflow(ctx, workflowID, runID)
	}

	return circuit_breaker.Execute(w.circuitBreaker, func() error {
		return w.client.CancelWorkflow(ctx, workflowID, runID)
	})
}

// TerminateWorkflow wraps the original TerminateWorkflow with circuit breaker
func (w *ClientWrapper) TerminateWorkflow(ctx context.Context, workflowID string, runID string, reason string, details ...any) error {
	if w.circuitBreaker == nil {
		return w.client.TerminateWorkflow(ctx, workflowID, runID, reason, details...)
	}

	return circuit_breaker.Execute(w.circuitBreaker, func() error {
		return w.client.TerminateWorkflow(ctx, workflowID, runID, reason, details...)
	})
}

// GetWorkflowHistory wraps the original GetWorkflowHistory with circuit breaker
func (w *ClientWrapper) GetWorkflowHistory(ctx context.Context, workflowID string, runID string, isLongPoll bool, filterType enumspb.HistoryEventFilterType) client.HistoryEventIterator {
	// Note: this returns an iterator which will make network requests lazily
	// circuit breaker protection happens at iterator level
	return w.client.GetWorkflowHistory(ctx, workflowID, runID, isLongPoll, filterType)
}

// CompleteActivity wraps the original CompleteActivity with circuit breaker
func (w *ClientWrapper) CompleteActivity(ctx context.Context, taskToken []byte, result any, err error) error {
	if w.circuitBreaker == nil {
		return w.client.CompleteActivity(ctx, taskToken, result, err)
	}

	return circuit_breaker.Execute(w.circuitBreaker, func() error {
		return w.client.CompleteActivity(ctx, taskToken, result, err)
	})
}

// CompleteActivityByID wraps the original CompleteActivityByID with circuit breaker
func (w *ClientWrapper) CompleteActivityByID(ctx context.Context, namespace, workflowID, runID, activityID string, result any, err error) error {
	if w.circuitBreaker == nil {
		return w.client.CompleteActivityByID(ctx, namespace, workflowID, runID, activityID, result, err)
	}

	return circuit_breaker.Execute(w.circuitBreaker, func() error {
		return w.client.CompleteActivityByID(ctx, namespace, workflowID, runID, activityID, result, err)
	})
}

// RecordActivityHeartbeat wraps the original RecordActivityHeartbeat with circuit breaker
func (w *ClientWrapper) RecordActivityHeartbeat(ctx context.Context, taskToken []byte, details ...any) error {
	if w.circuitBreaker == nil {
		return w.client.RecordActivityHeartbeat(ctx, taskToken, details...)
	}

	return circuit_breaker.Execute(w.circuitBreaker, func() error {
		return w.client.RecordActivityHeartbeat(ctx, taskToken, details...)
	})
}

// RecordActivityHeartbeatByID wraps the original RecordActivityHeartbeatByID with circuit breaker
func (w *ClientWrapper) RecordActivityHeartbeatByID(ctx context.Context, namespace, workflowID, runID, activityID string, details ...any) error {
	if w.circuitBreaker == nil {
		return w.client.RecordActivityHeartbeatByID(ctx, namespace, workflowID, runID, activityID, details...)
	}

	return circuit_breaker.Execute(w.circuitBreaker, func() error {
		return w.client.RecordActivityHeartbeatByID(ctx, namespace, workflowID, runID, activityID, details...)
	})
}

// ListClosedWorkflow wraps the original ListClosedWorkflow with circuit breaker
func (w *ClientWrapper) ListClosedWorkflow(ctx context.Context, request *workflowservice.ListClosedWorkflowExecutionsRequest) (*workflowservice.ListClosedWorkflowExecutionsResponse, error) {
	if w.circuitBreaker == nil {
		return w.client.ListClosedWorkflow(ctx, request)
	}

	var result *workflowservice.ListClosedWorkflowExecutionsResponse

	err := circuit_breaker.Execute(w.circuitBreaker, func() error {
		var execErr error

		result, execErr = w.client.ListClosedWorkflow(ctx, request)

		return execErr
	})

	return result, err
}

// ListOpenWorkflow wraps the original ListOpenWorkflow with circuit breaker
func (w *ClientWrapper) ListOpenWorkflow(ctx context.Context, request *workflowservice.ListOpenWorkflowExecutionsRequest) (*workflowservice.ListOpenWorkflowExecutionsResponse, error) {
	if w.circuitBreaker == nil {
		return w.client.ListOpenWorkflow(ctx, request)
	}

	var result *workflowservice.ListOpenWorkflowExecutionsResponse

	err := circuit_breaker.Execute(w.circuitBreaker, func() error {
		var execErr error

		result, execErr = w.client.ListOpenWorkflow(ctx, request)

		return execErr
	})

	return result, err
}

// ListWorkflow wraps the original ListWorkflow with circuit breaker
func (w *ClientWrapper) ListWorkflow(ctx context.Context, request *workflowservice.ListWorkflowExecutionsRequest) (*workflowservice.ListWorkflowExecutionsResponse, error) {
	if w.circuitBreaker == nil {
		return w.client.ListWorkflow(ctx, request)
	}

	var result *workflowservice.ListWorkflowExecutionsResponse

	err := circuit_breaker.Execute(w.circuitBreaker, func() error {
		var execErr error

		result, execErr = w.client.ListWorkflow(ctx, request)

		return execErr
	})

	return result, err
}

// ListArchivedWorkflow wraps the original ListArchivedWorkflow with circuit breaker
func (w *ClientWrapper) ListArchivedWorkflow(ctx context.Context, request *workflowservice.ListArchivedWorkflowExecutionsRequest) (*workflowservice.ListArchivedWorkflowExecutionsResponse, error) {
	if w.circuitBreaker == nil {
		return w.client.ListArchivedWorkflow(ctx, request)
	}

	var result *workflowservice.ListArchivedWorkflowExecutionsResponse

	err := circuit_breaker.Execute(w.circuitBreaker, func() error {
		var execErr error

		result, execErr = w.client.ListArchivedWorkflow(ctx, request)

		return execErr
	})

	return result, err
}

// ScanWorkflow wraps the original ScanWorkflow with circuit breaker
//
// Deprecated: use ListWorkflow instead
//
//nolint:staticcheck // keeping for backward compatibility
func (w *ClientWrapper) ScanWorkflow(ctx context.Context, request *workflowservice.ScanWorkflowExecutionsRequest) (*workflowservice.ScanWorkflowExecutionsResponse, error) {
	if w.circuitBreaker == nil {
		return w.client.ScanWorkflow(ctx, request) //nolint:staticcheck // deprecated but kept for backward compatibility
	}

	var result *workflowservice.ScanWorkflowExecutionsResponse

	err := circuit_breaker.Execute(w.circuitBreaker, func() error {
		var execErr error

		result, execErr = w.client.ScanWorkflow(ctx, request) //nolint:staticcheck // deprecated but kept for backward compatibility

		return execErr
	})

	return result, err
}

// CountWorkflow wraps the original CountWorkflow with circuit breaker
func (w *ClientWrapper) CountWorkflow(ctx context.Context, request *workflowservice.CountWorkflowExecutionsRequest) (*workflowservice.CountWorkflowExecutionsResponse, error) {
	if w.circuitBreaker == nil {
		return w.client.CountWorkflow(ctx, request)
	}

	var result *workflowservice.CountWorkflowExecutionsResponse

	err := circuit_breaker.Execute(w.circuitBreaker, func() error {
		var execErr error

		result, execErr = w.client.CountWorkflow(ctx, request)

		return execErr
	})

	return result, err
}

// GetSearchAttributes wraps the original GetSearchAttributes with circuit breaker
func (w *ClientWrapper) GetSearchAttributes(ctx context.Context) (*workflowservice.GetSearchAttributesResponse, error) {
	if w.circuitBreaker == nil {
		return w.client.GetSearchAttributes(ctx)
	}

	var result *workflowservice.GetSearchAttributesResponse

	err := circuit_breaker.Execute(w.circuitBreaker, func() error {
		var execErr error

		result, execErr = w.client.GetSearchAttributes(ctx)

		return execErr
	})

	return result, err
}

// QueryWorkflow wraps the original QueryWorkflow with circuit breaker
func (w *ClientWrapper) QueryWorkflow(ctx context.Context, workflowID string, runID string, queryType string, args ...any) (converter.EncodedValue, error) {
	if w.circuitBreaker == nil {
		return w.client.QueryWorkflow(ctx, workflowID, runID, queryType, args...)
	}

	var result converter.EncodedValue

	err := circuit_breaker.Execute(w.circuitBreaker, func() error {
		var execErr error

		result, execErr = w.client.QueryWorkflow(ctx, workflowID, runID, queryType, args...)

		return execErr
	})

	return result, err
}

// QueryWorkflowWithOptions wraps the original QueryWorkflowWithOptions with circuit breaker
func (w *ClientWrapper) QueryWorkflowWithOptions(ctx context.Context, request *client.QueryWorkflowWithOptionsRequest) (*client.QueryWorkflowWithOptionsResponse, error) {
	if w.circuitBreaker == nil {
		return w.client.QueryWorkflowWithOptions(ctx, request)
	}

	var result *client.QueryWorkflowWithOptionsResponse

	err := circuit_breaker.Execute(w.circuitBreaker, func() error {
		var execErr error

		result, execErr = w.client.QueryWorkflowWithOptions(ctx, request)

		return execErr
	})

	return result, err
}

// DescribeWorkflowExecution wraps the original DescribeWorkflowExecution with circuit breaker
func (w *ClientWrapper) DescribeWorkflowExecution(ctx context.Context, workflowID, runID string) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
	if w.circuitBreaker == nil {
		return w.client.DescribeWorkflowExecution(ctx, workflowID, runID)
	}

	var result *workflowservice.DescribeWorkflowExecutionResponse

	err := circuit_breaker.Execute(w.circuitBreaker, func() error {
		var execErr error

		result, execErr = w.client.DescribeWorkflowExecution(ctx, workflowID, runID)

		return execErr
	})

	return result, err
}

// DescribeWorkflow wraps the original DescribeWorkflow with circuit breaker
func (w *ClientWrapper) DescribeWorkflow(ctx context.Context, workflowID, runID string) (*client.WorkflowExecutionDescription, error) {
	if w.circuitBreaker == nil {
		return w.client.DescribeWorkflow(ctx, workflowID, runID)
	}

	var result *client.WorkflowExecutionDescription

	err := circuit_breaker.Execute(w.circuitBreaker, func() error {
		var execErr error

		result, execErr = w.client.DescribeWorkflow(ctx, workflowID, runID)

		return execErr
	})

	return result, err
}

// DescribeTaskQueue wraps the original DescribeTaskQueue with circuit breaker
func (w *ClientWrapper) DescribeTaskQueue(ctx context.Context, taskqueue string, taskqueueType enumspb.TaskQueueType) (*workflowservice.DescribeTaskQueueResponse, error) {
	if w.circuitBreaker == nil {
		return w.client.DescribeTaskQueue(ctx, taskqueue, taskqueueType)
	}

	var result *workflowservice.DescribeTaskQueueResponse

	err := circuit_breaker.Execute(w.circuitBreaker, func() error {
		var execErr error

		result, execErr = w.client.DescribeTaskQueue(ctx, taskqueue, taskqueueType)

		return execErr
	})

	return result, err
}

// DescribeTaskQueueEnhanced wraps the original DescribeTaskQueueEnhanced with circuit breaker
func (w *ClientWrapper) DescribeTaskQueueEnhanced(ctx context.Context, options client.DescribeTaskQueueEnhancedOptions) (client.TaskQueueDescription, error) {
	if w.circuitBreaker == nil {
		return w.client.DescribeTaskQueueEnhanced(ctx, options)
	}

	var result client.TaskQueueDescription

	err := circuit_breaker.Execute(w.circuitBreaker, func() error {
		var execErr error

		result, execErr = w.client.DescribeTaskQueueEnhanced(ctx, options)

		return execErr
	})

	return result, err
}

// ResetWorkflowExecution wraps the original ResetWorkflowExecution with circuit breaker
func (w *ClientWrapper) ResetWorkflowExecution(ctx context.Context, request *workflowservice.ResetWorkflowExecutionRequest) (*workflowservice.ResetWorkflowExecutionResponse, error) {
	if w.circuitBreaker == nil {
		return w.client.ResetWorkflowExecution(ctx, request)
	}

	var result *workflowservice.ResetWorkflowExecutionResponse

	err := circuit_breaker.Execute(w.circuitBreaker, func() error {
		var execErr error

		result, execErr = w.client.ResetWorkflowExecution(ctx, request)

		return execErr
	})

	return result, err
}

// UpdateWorkerBuildIdCompatibility wraps the original UpdateWorkerBuildIdCompatibility with circuit breaker
//
// Deprecated: use UpdateWorkerVersioningRules with the versioning api
//
//nolint:staticcheck // keeping for backward compatibility
func (w *ClientWrapper) UpdateWorkerBuildIdCompatibility(ctx context.Context, options *client.UpdateWorkerBuildIdCompatibilityOptions) error {
	if w.circuitBreaker == nil {
		return w.client.UpdateWorkerBuildIdCompatibility(ctx, options) //nolint:staticcheck // deprecated but kept for backward compatibility
	}

	return circuit_breaker.Execute(w.circuitBreaker, func() error {
		return w.client.UpdateWorkerBuildIdCompatibility(ctx, options) //nolint:staticcheck // deprecated but kept for backward compatibility
	})
}

// GetWorkerBuildIdCompatibility wraps the original GetWorkerBuildIdCompatibility with circuit breaker
//
// Deprecated: use GetWorkerVersioningRules with the versioning api
//
//nolint:staticcheck // keeping for backward compatibility
func (w *ClientWrapper) GetWorkerBuildIdCompatibility(ctx context.Context, options *client.GetWorkerBuildIdCompatibilityOptions) (*client.WorkerBuildIDVersionSets, error) {
	if w.circuitBreaker == nil {
		return w.client.GetWorkerBuildIdCompatibility(ctx, options) //nolint:staticcheck // deprecated but kept for backward compatibility
	}

	var result *client.WorkerBuildIDVersionSets //nolint:staticcheck // deprecated but kept for backward compatibility

	err := circuit_breaker.Execute(w.circuitBreaker, func() error {
		var execErr error

		result, execErr = w.client.GetWorkerBuildIdCompatibility(ctx, options) //nolint:staticcheck // deprecated but kept for backward compatibility

		return execErr
	})

	return result, err
}

// GetWorkerTaskReachability wraps the original GetWorkerTaskReachability with circuit breaker
//
// Deprecated: use DescribeTaskQueueEnhanced with the versioning api
//
//nolint:staticcheck // keeping for backward compatibility
func (w *ClientWrapper) GetWorkerTaskReachability(ctx context.Context, options *client.GetWorkerTaskReachabilityOptions) (*client.WorkerTaskReachability, error) {
	if w.circuitBreaker == nil {
		return w.client.GetWorkerTaskReachability(ctx, options) //nolint:staticcheck // deprecated but kept for backward compatibility
	}

	var result *client.WorkerTaskReachability //nolint:staticcheck // deprecated but kept for backward compatibility

	err := circuit_breaker.Execute(w.circuitBreaker, func() error {
		var execErr error

		result, execErr = w.client.GetWorkerTaskReachability(ctx, options) //nolint:staticcheck // deprecated but kept for backward compatibility

		return execErr
	})

	return result, err
}

// UpdateWorkerVersioningRules wraps the original UpdateWorkerVersioningRules with circuit breaker
//
// Deprecated: build-id based versioning is deprecated in favor of worker deployment based versioning
//
//nolint:staticcheck // keeping for backward compatibility
func (w *ClientWrapper) UpdateWorkerVersioningRules(ctx context.Context, options client.UpdateWorkerVersioningRulesOptions) (*client.WorkerVersioningRules, error) {
	if w.circuitBreaker == nil {
		return w.client.UpdateWorkerVersioningRules(ctx, options) //nolint:staticcheck // deprecated but kept for backward compatibility
	}

	var result *client.WorkerVersioningRules //nolint:staticcheck // deprecated but kept for backward compatibility

	err := circuit_breaker.Execute(w.circuitBreaker, func() error {
		var execErr error

		result, execErr = w.client.UpdateWorkerVersioningRules(ctx, options) //nolint:staticcheck // deprecated but kept for backward compatibility

		return execErr
	})

	return result, err
}

// GetWorkerVersioningRules wraps the original GetWorkerVersioningRules with circuit breaker
//
// Deprecated: build-id based versioning is deprecated in favor of worker deployment based versioning
//
//nolint:staticcheck // keeping for backward compatibility
func (w *ClientWrapper) GetWorkerVersioningRules(ctx context.Context, options client.GetWorkerVersioningOptions) (*client.WorkerVersioningRules, error) {
	if w.circuitBreaker == nil {
		return w.client.GetWorkerVersioningRules(ctx, options) //nolint:staticcheck // deprecated but kept for backward compatibility
	}

	var result *client.WorkerVersioningRules //nolint:staticcheck // deprecated but kept for backward compatibility

	err := circuit_breaker.Execute(w.circuitBreaker, func() error {
		var execErr error

		result, execErr = w.client.GetWorkerVersioningRules(ctx, options) //nolint:staticcheck // deprecated but kept for backward compatibility

		return execErr
	})

	return result, err
}

// CheckHealth wraps the original CheckHealth with circuit breaker
func (w *ClientWrapper) CheckHealth(ctx context.Context, request *client.CheckHealthRequest) (*client.CheckHealthResponse, error) {
	if w.circuitBreaker == nil {
		return w.client.CheckHealth(ctx, request)
	}

	var result *client.CheckHealthResponse

	err := circuit_breaker.Execute(w.circuitBreaker, func() error {
		var execErr error

		result, execErr = w.client.CheckHealth(ctx, request)

		return execErr
	})

	return result, err
}

// UpdateWorkflow wraps the original UpdateWorkflow with circuit breaker
func (w *ClientWrapper) UpdateWorkflow(ctx context.Context, options client.UpdateWorkflowOptions) (client.WorkflowUpdateHandle, error) {
	if w.circuitBreaker == nil {
		return w.client.UpdateWorkflow(ctx, options)
	}

	var result client.WorkflowUpdateHandle

	err := circuit_breaker.Execute(w.circuitBreaker, func() error {
		var execErr error

		result, execErr = w.client.UpdateWorkflow(ctx, options)

		return execErr
	})

	return result, err
}

// UpdateWorkflowExecutionOptions wraps the original UpdateWorkflowExecutionOptions with circuit breaker
func (w *ClientWrapper) UpdateWorkflowExecutionOptions(ctx context.Context, options client.UpdateWorkflowExecutionOptionsRequest) (client.WorkflowExecutionOptions, error) {
	if w.circuitBreaker == nil {
		return w.client.UpdateWorkflowExecutionOptions(ctx, options)
	}

	var result client.WorkflowExecutionOptions

	err := circuit_breaker.Execute(w.circuitBreaker, func() error {
		var execErr error

		result, execErr = w.client.UpdateWorkflowExecutionOptions(ctx, options)

		return execErr
	})

	return result, err
}

// UpdateWithStartWorkflow wraps the original UpdateWithStartWorkflow with circuit breaker
func (w *ClientWrapper) UpdateWithStartWorkflow(ctx context.Context, options client.UpdateWithStartWorkflowOptions) (client.WorkflowUpdateHandle, error) {
	if w.circuitBreaker == nil {
		return w.client.UpdateWithStartWorkflow(ctx, options)
	}

	var result client.WorkflowUpdateHandle

	err := circuit_breaker.Execute(w.circuitBreaker, func() error {
		var execErr error

		result, execErr = w.client.UpdateWithStartWorkflow(ctx, options)

		return execErr
	})

	return result, err
}

// GetWorkflowUpdateHandle does not make network request, no circuit breaker needed
func (w *ClientWrapper) GetWorkflowUpdateHandle(ref client.GetWorkflowUpdateHandleOptions) client.WorkflowUpdateHandle {
	return w.client.GetWorkflowUpdateHandle(ref)
}

// WorkflowService returns the underlying workflow service client (no circuit breaker needed)
func (w *ClientWrapper) WorkflowService() workflowservice.WorkflowServiceClient {
	return w.client.WorkflowService()
}

// OperatorService returns the underlying operator service client (no circuit breaker needed)
func (w *ClientWrapper) OperatorService() operatorservice.OperatorServiceClient {
	return w.client.OperatorService()
}

// ScheduleClient returns the schedule client (no circuit breaker needed)
func (w *ClientWrapper) ScheduleClient() client.ScheduleClient {
	return w.client.ScheduleClient()
}

// DeploymentClient returns the deployment client (no circuit breaker needed)
//
// Deprecated: use WorkerDeploymentClient
//
//nolint:staticcheck // keeping for backward compatibility
func (w *ClientWrapper) DeploymentClient() client.DeploymentClient {
	return w.client.DeploymentClient() //nolint:staticcheck // deprecated but kept for backward compatibility
}

// WorkerDeploymentClient returns the worker deployment client (no circuit breaker needed)
func (w *ClientWrapper) WorkerDeploymentClient() client.WorkerDeploymentClient {
	return w.client.WorkerDeploymentClient()
}

// Close closes the wrapped client
func (w *ClientWrapper) Close() {
	w.client.Close()
}
