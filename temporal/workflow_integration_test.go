package temporal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.uber.org/mock/gomock"

	mockTemporal "github.com/pixality-inc/golang-core/mocks/temporal"
)

func TestWorkflowImpl_WithRealWorkerImpl(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mockTemporal.NewMockClient(ctrl)

	workerImpl := &WorkerImpl{
		client: mockClient,
		worker: nil,
	}

	cfg := WorkflowConfig{
		Name: "OrderWorkflow",
	}

	workFlow := NewWorkflowImpl(workerImpl, cfg)

	ctx := t.Context()
	workflowID := "order-42"
	queue := "orders"
	input := map[string]any{"user_id": 7}

	expectedOpts := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: queue,
	}

	var run client.WorkflowRun

	mockClient.EXPECT().
		ExecuteWorkflow(ctx, expectedOpts, WorkflowName("OrderWorkflow"), input).
		Return(run, nil)

	res, err := workFlow.Apply(ctx, workflowID, queue, input)
	require.NoError(t, err)
	require.Equal(t, run, res)
}

func TestWorkflowImpl_WithRealWorkerImpl_Error(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mockTemporal.NewMockClient(ctrl)

	workerImpl := &WorkerImpl{
		client: mockClient,
	}

	cfg := WorkflowConfig{Name: "FailWorkflow"}
	workFlow := NewWorkflowImpl(workerImpl, cfg)

	expectedErr := assert.AnError

	mockClient.EXPECT().
		ExecuteWorkflow(gomock.Any(), gomock.Any(), WorkflowName("FailWorkflow"), gomock.Any()).
		Return(nil, expectedErr)

	res, err := workFlow.Apply(t.Context(), "id", "queue", "input")
	require.ErrorIs(t, err, expectedErr)
	require.Nil(t, res)
}
