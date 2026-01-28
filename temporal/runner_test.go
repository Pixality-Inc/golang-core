package temporal_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/mock/gomock"

	"github.com/pixality-inc/golang-core/temporal"
	mockTemporal "github.com/pixality-inc/golang-core/temporal/mocks"
)

var (
	errFutureExecution = errors.New("future execution failed")
	errChannel         = errors.New("channel error")
	errActivity        = errors.New("activity failed")
)

func TestWaitForFutures_Success(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	mockFuture := mockTemporal.NewMockActivityFuture(ctrl)
	mockFuture.EXPECT().IsReady().Return(true).AnyTimes()
	mockFuture.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil)

	mockAwaitable := mockTemporal.NewMockAwaitable(ctrl)
	mockAwaitable.EXPECT().Name().Return("test-future").AnyTimes()
	mockAwaitable.EXPECT().Future().Return(mockFuture).AnyTimes()

	env.RegisterWorkflow(func(ctx workflow.Context) error {
		err := temporal.WaitForFutures(ctx, time.Second, mockAwaitable)

		return err
	})

	env.ExecuteWorkflow(func(ctx workflow.Context) error {
		err := temporal.WaitForFutures(ctx, time.Second, mockAwaitable)

		return err
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

func TestWaitForFutures_Timeout(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	mockFuture := mockTemporal.NewMockActivityFuture(ctrl)
	mockFuture.EXPECT().IsReady().Return(false).AnyTimes()

	mockAwaitable := mockTemporal.NewMockAwaitable(ctrl)
	mockAwaitable.EXPECT().Name().Return("test-future").AnyTimes()
	mockAwaitable.EXPECT().Future().Return(mockFuture).AnyTimes()

	env.RegisterWorkflow(func(ctx workflow.Context) error {
		err := temporal.WaitForFutures(ctx, time.Millisecond, mockAwaitable)

		return err
	})

	env.ExecuteWorkflow(func(ctx workflow.Context) error {
		err := temporal.WaitForFutures(ctx, time.Millisecond, mockAwaitable)

		return err
	})

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
}

func TestWaitForFutures_FutureError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	mockFuture := mockTemporal.NewMockActivityFuture(ctrl)
	mockFuture.EXPECT().IsReady().Return(true).AnyTimes()
	mockFuture.EXPECT().Get(gomock.Any(), gomock.Any()).Return(errFutureExecution)

	mockAwaitable := mockTemporal.NewMockAwaitable(ctrl)
	mockAwaitable.EXPECT().Name().Return("failed-future").AnyTimes()
	mockAwaitable.EXPECT().Future().Return(mockFuture).AnyTimes()

	env.RegisterWorkflow(func(ctx workflow.Context) error {
		err := temporal.WaitForFutures(ctx, time.Second, mockAwaitable)

		return err
	})

	env.ExecuteWorkflow(func(ctx workflow.Context) error {
		err := temporal.WaitForFutures(ctx, time.Second, mockAwaitable)

		return err
	})

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
}

func TestWaitForFutures_MultipleFutures(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	mockFuture1 := mockTemporal.NewMockActivityFuture(ctrl)
	mockFuture1.EXPECT().IsReady().Return(true).AnyTimes()
	mockFuture1.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil)

	mockAwaitable1 := mockTemporal.NewMockAwaitable(ctrl)
	mockAwaitable1.EXPECT().Name().Return("future-1").AnyTimes()
	mockAwaitable1.EXPECT().Future().Return(mockFuture1).AnyTimes()

	mockFuture2 := mockTemporal.NewMockActivityFuture(ctrl)
	mockFuture2.EXPECT().IsReady().Return(true).AnyTimes()
	mockFuture2.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil)

	mockAwaitable2 := mockTemporal.NewMockAwaitable(ctrl)
	mockAwaitable2.EXPECT().Name().Return("future-2").AnyTimes()
	mockAwaitable2.EXPECT().Future().Return(mockFuture2).AnyTimes()

	env.RegisterWorkflow(func(ctx workflow.Context) error {
		err := temporal.WaitForFutures(ctx, time.Second, mockAwaitable1, mockAwaitable2)

		return err
	})

	env.ExecuteWorkflow(func(ctx workflow.Context) error {
		err := temporal.WaitForFutures(ctx, time.Second, mockAwaitable1, mockAwaitable2)

		return err
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

func TestWaitForChannels_NoChannels(t *testing.T) {
	t.Parallel()

	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(func(ctx workflow.Context) error {
		return temporal.WaitForChannels(ctx)
	})

	env.ExecuteWorkflow(func(ctx workflow.Context) error {
		return temporal.WaitForChannels(ctx)
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

func TestWaitForChannels_Success(t *testing.T) {
	t.Parallel()

	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(func(ctx workflow.Context) error {
		ch := workflow.NewChannel(ctx)

		workflow.Go(ctx, func(ctx workflow.Context) {
			ch.Send(ctx, nil)
		})

		return temporal.WaitForChannels(ctx, ch)
	})

	env.ExecuteWorkflow(func(ctx workflow.Context) error {
		ch := workflow.NewChannel(ctx)

		workflow.Go(ctx, func(ctx workflow.Context) {
			ch.Send(ctx, nil)
		})

		return temporal.WaitForChannels(ctx, ch)
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

func TestWaitForChannels_WithErrors(t *testing.T) {
	t.Parallel()

	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(func(ctx workflow.Context) error {
		ch := workflow.NewChannel(ctx)

		workflow.Go(ctx, func(ctx workflow.Context) {
			ch.Send(ctx, errChannel)
		})

		return temporal.WaitForChannels(ctx, ch)
	})

	env.ExecuteWorkflow(func(ctx workflow.Context) error {
		ch := workflow.NewChannel(ctx)

		workflow.Go(ctx, func(ctx workflow.Context) {
			ch.Send(ctx, errActivity)
		})

		return temporal.WaitForChannels(ctx, ch)
	})

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
}

func TestExecuteActivitySync_Error(t *testing.T) {
	t.Parallel()

	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	activity := temporal.NewActivityImpl(nil, temporal.ActivityConfig{
		Name:                    "failing-activity",
		Queue:                   "test-queue",
		Timeout:                 time.Minute,
		MaxAttempts:             1,
		RetryInitialInterval:    time.Second,
		RetryBackoffCoefficient: 2.0,
		RetryMaximumInterval:    time.Minute,
	})

	runner := func(ctx context.Context, input string) (string, error) {
		return "", errActivity
	}

	wrapper := temporal.NewActivityTypedWrapper(activity, runner)

	env.RegisterActivity(runner)

	env.ExecuteWorkflow(func(ctx workflow.Context) (string, error) {
		return temporal.ExecuteActivitySync(ctx, wrapper, "default", "test-input")
	})

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
}
