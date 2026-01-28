//go:generate mockgen -destination=../mocks/temporal/client_gen.go -package=mock_temporal go.temporal.io/sdk/client Client
//go:generate mockgen -destination=../mocks/temporal/worker_gen.go -package=mock_temporal go.temporal.io/sdk/worker Worker

package temporal

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/mock/gomock"

	"github.com/pixality-inc/golang-core/logger"
	mockTemporal "github.com/pixality-inc/golang-core/mocks/temporal"
)

type testActivity struct{}

func (t testActivity) Queue() QueueName {
	return "TestActivity"
}

func (t testActivity) Timeout() time.Duration {
	return time.Second
}

func (t testActivity) MaxAttempts() int {
	return 5
}

func (t testActivity) RetryInitialInterval() time.Duration {
	return time.Second
}

func (t testActivity) RetryBackoffCoefficient() float64 {
	return 0.1
}

func (t testActivity) RetryMaximumInterval() time.Duration {
	return time.Second
}

func (t testActivity) Name() ActivityName { return "TestActivity" }

type testWorkflow struct{}

func (t testWorkflow) Name() WorkflowName { return "TestWorkflow" }

func (t testWorkflow) Apply(_ context.Context, _ string, _ string, _ any) (client.WorkflowRun, error) {
	return nil, nil
}

func newTestWorker(
	mockClient *mockTemporal.MockClient,
	mockWorker *mockTemporal.MockWorker,
) *WorkerImpl {
	return &WorkerImpl{
		client: mockClient,
		worker: mockWorker,
		log:    logger.NewLoggableImpl(nil),
	}
}

func TestWorker_Run(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mockTemporal.NewMockClient(ctrl)
	mockWorker := mockTemporal.NewMockWorker(ctrl)

	w := &WorkerImpl{
		client: mockClient,
		worker: mockWorker,
	}

	mockWorker.EXPECT().
		Run(nil).
		Return(nil)

	err := w.Run()
	require.NoError(t, err)
}

func TestWorker_Stop(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mockTemporal.NewMockClient(ctrl)
	mockWorker := mockTemporal.NewMockWorker(ctrl)

	w := newTestWorker(mockClient, mockWorker)

	mockWorker.EXPECT().Stop()

	w.Stop()
}

func TestRegisterWorkflow(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mockTemporal.NewMockClient(ctrl)
	mockWorker := mockTemporal.NewMockWorker(ctrl)

	w := newTestWorker(mockClient, mockWorker)

	runner := func(ctx workflow.Context) error { return nil }

	mockWorker.EXPECT().
		RegisterWorkflowWithOptions(
			gomock.AssignableToTypeOf(func(workflow.Context) error { return nil }),
			workflow.RegisterOptions{Name: "TestWorkflow"},
		)

	err := w.RegisterWorkflow(testWorkflow{}, runner)
	require.NoError(t, err)
}

func TestRegisterActivity(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mockTemporal.NewMockClient(ctrl)
	mockWorker := mockTemporal.NewMockWorker(ctrl)

	w := newTestWorker(mockClient, mockWorker)

	runner := func(ctx context.Context) error { return nil }

	mockWorker.EXPECT().
		RegisterActivityWithOptions(
			gomock.AssignableToTypeOf(func(context.Context) error { return nil }),
			activity.RegisterOptions{Name: "TestActivity"},
		)

	err := w.RegisterActivity(testActivity{}, runner)
	require.NoError(t, err)
}

func TestWorker_ExecuteWorkflow(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mockTemporal.NewMockClient(ctrl)
	mockWorker := mockTemporal.NewMockWorker(ctrl)

	w := newTestWorker(mockClient, mockWorker)

	ctx := context.Background()
	opts := client.StartWorkflowOptions{
		ID:        "wf-id",
		TaskQueue: "queue",
	}

	var run client.WorkflowRun

	mockClient.EXPECT().
		ExecuteWorkflow(ctx, opts, "wfName", 1, 2).
		Return(run, nil)

	res, err := w.ExecuteWorkflow(ctx, opts, "wfName", 1, 2)
	require.NoError(t, err)
	require.Equal(t, run, res)
}
