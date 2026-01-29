package temporal_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.uber.org/mock/gomock"

	"github.com/pixality-inc/golang-core/temporal"
	mockTemporal "github.com/pixality-inc/golang-core/temporal/mocks"
)

func TestWorkflowImpl_Name(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWorker := mockTemporal.NewMockWorker(ctrl)

	cfg := temporal.WorkflowConfig{
		Name: "MyWorkflow",
	}

	wf := temporal.NewWorkflowImpl(mockWorker, cfg)

	require.Equal(t, temporal.WorkflowName("MyWorkflow"), wf.Name())
}

func TestWorkflowImpl_Apply(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWorker := mockTemporal.NewMockWorker(ctrl)

	cfg := temporal.WorkflowConfig{
		Name: "PaymentWorkflow",
	}

	workFlow := temporal.NewWorkflowImpl(mockWorker, cfg)

	ctx := t.Context()
	workflowID := "wf-123"
	queue := "critical-queue"
	input := map[string]any{"amount": 42}

	opts := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: queue,
	}

	var run client.WorkflowRun

	mockWorker.EXPECT().
		ExecuteWorkflow(ctx, opts, temporal.WorkflowName("PaymentWorkflow"), input).
		Return(run, nil)

	res, err := workFlow.Apply(ctx, workflowID, queue, input)
	require.NoError(t, err)
	require.Equal(t, run, res)
}

func TestWorkflowImpl_Apply_Error(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWorker := mockTemporal.NewMockWorker(ctrl)

	cfg := temporal.WorkflowConfig{
		Name: "FailingWorkflow",
	}

	workFlow := temporal.NewWorkflowImpl(mockWorker, cfg)

	ctx := t.Context()
	expectedErr := assert.AnError

	mockWorker.EXPECT().
		ExecuteWorkflow(gomock.Any(), gomock.Any(), temporal.WorkflowName("FailingWorkflow"), gomock.Any()).
		Return(nil, expectedErr)

	res, err := workFlow.Apply(ctx, "id", "queue", "input")
	require.ErrorIs(t, err, expectedErr)
	require.Nil(t, res)
}

func TestWorkflowImpl_GetLoggerWithoutContext(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWorker := mockTemporal.NewMockWorker(ctrl)

	cfg := temporal.WorkflowConfig{Name: "LoggerWorkflow"}

	wf := temporal.NewWorkflowImpl(mockWorker, cfg)

	logger := wf.GetLoggerWithoutContext()
	require.NotNil(t, logger)
}
