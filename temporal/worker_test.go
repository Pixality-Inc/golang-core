//go:generate mockgen -destination=../mocks/temporal/client_gen.go -package=mock_temporal go.temporal.io/sdk/client Client
//go:generate mockgen -destination=../mocks/temporal/worker_gen.go -package=mock_temporal go.temporal.io/sdk/worker Worker

package temporal

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
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

func (t testActivity) HeartbeatTimeout() time.Duration {
	return 0
}

func (t testActivity) ScheduleToCloseTimeout() time.Duration {
	return 0
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

func (t testWorkflow) Apply(_ context.Context, _ string, _ string, _ any, _ ...StartWorkflowOption) (client.WorkflowRun, error) {
	return nil, nil
}

func newTestWorker(
	mockClient *mockTemporal.MockClient,
	mockWorker *mockTemporal.MockWorker,
) *WorkerImpl {
	return &WorkerImpl{
		client:     mockClient,
		worker:     mockWorker,
		log:        logger.NewLoggableImpl(nil),
		heartbeats: &heartbeatInterceptor{},
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

type heartbeatTestActivity struct {
	testActivity
}

func (t heartbeatTestActivity) HeartbeatTimeout() time.Duration {
	return time.Second
}

// heartbeatTestTimeout is the heartbeat timeout the test workflow schedules the activity with. the
// test env times an activity out on it in real time, and the sdk sends a heartbeat at most every
// 0.8 of it, so it is long enough for a loaded runner to keep up
const heartbeatTestTimeout = time.Second

// runHeartbeatTestActivity registers the activity on w, runs it for runFor in a workflow that
// schedules it with heartbeatTimeout and one attempt, and returns its error and how many heartbeats
// it sent
func runHeartbeatTestActivity(
	t *testing.T,
	activityConfig Activity,
	heartbeatTimeout time.Duration,
	runFor time.Duration,
) (error, int32) {
	t.Helper()

	ctrl := gomock.NewController(t)
	mockWorker := mockTemporal.NewMockWorker(ctrl)

	w := newTestWorker(mockTemporal.NewMockClient(ctrl), mockWorker)

	runner := func(ctx context.Context) error {
		select {
		case <-time.After(runFor):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	var registered any

	mockWorker.EXPECT().
		RegisterActivityWithOptions(gomock.AssignableToTypeOf(runner), activity.RegisterOptions{Name: "TestActivity"}).
		Do(func(r any, _ activity.RegisterOptions) { registered = r })

	require.NoError(t, w.RegisterActivity(activityConfig, runner))

	// the test env reports a heartbeat timeout once the activity returns, so it has to outlast the
	// default idle timeout of the env
	env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()
	env.SetTestTimeout(time.Minute)
	env.SetWorkerOptions(worker.Options{Interceptors: []interceptor.WorkerInterceptor{w.heartbeats}})

	var beats atomic.Int32

	env.SetOnActivityHeartbeatListener(func(*activity.Info, converter.EncodedValues) { beats.Add(1) })
	env.RegisterActivityWithOptions(registered, activity.RegisterOptions{Name: "TestActivity"})
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context) error {
		ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: time.Minute,
			HeartbeatTimeout:    heartbeatTimeout,
			RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 1},
		})

		return workflow.ExecuteActivity(ctx, "TestActivity").Get(ctx, nil)
	}, workflow.RegisterOptions{Name: "TestHeartbeatWorkflow"})

	env.ExecuteWorkflow("TestHeartbeatWorkflow")
	require.True(t, env.IsWorkflowCompleted())

	return env.GetWorkflowError(), beats.Load()
}

func TestRegisterActivity_HeartbeatKeepsALongActivityAlive(t *testing.T) {
	t.Parallel()

	err, beats := runHeartbeatTestActivity(t, heartbeatTestActivity{}, heartbeatTestTimeout, 3*heartbeatTestTimeout)
	require.NoError(t, err)
	require.Positive(t, beats)
}

// the activity config of the worker opts in: an activity registered without a heartbeat timeout is
// not heartbeaten for, even when its scheduler asks for heartbeats
func TestRegisterActivity_NoHeartbeatWithoutOptIn(t *testing.T) {
	t.Parallel()

	err, beats := runHeartbeatTestActivity(t, testActivity{}, heartbeatTestTimeout, 2*heartbeatTestTimeout)

	var timeoutErr *temporal.TimeoutError

	require.ErrorAs(t, err, &timeoutErr)
	require.Equal(t, enumspb.TIMEOUT_TYPE_HEARTBEAT, timeoutErr.TimeoutType())
	require.Zero(t, beats)
}

// the interval follows the timeout the activity was scheduled with: none was asked for, none is sent
func TestRegisterActivity_NoHeartbeatWhenScheduledWithout(t *testing.T) {
	t.Parallel()

	err, beats := runHeartbeatTestActivity(t, heartbeatTestActivity{}, 0, heartbeatTestTimeout)
	require.NoError(t, err)
	require.Zero(t, beats)
}

func TestNewWorkerOptions_KeepsTheHeartbeatInterceptor(t *testing.T) {
	t.Parallel()

	heartbeats := &heartbeatInterceptor{}
	other := &interceptor.WorkerInterceptorBase{}

	options := newWorkerOptions(context.Background(), "worker", WorkerConfig{}, heartbeats, []WorkerOption{
		func(options *worker.Options) { options.Interceptors = []interceptor.WorkerInterceptor{other} },
	})

	require.Equal(t, []interceptor.WorkerInterceptor{other, heartbeats}, options.Interceptors)
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
