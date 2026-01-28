package temporal_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/mock/gomock"

	"github.com/pixality-inc/golang-core/temporal"
	mockTemporal "github.com/pixality-inc/golang-core/temporal/mocks"
)

//go:generate mockgen -destination mocks/activity_future_gen.go -source future_test.go -package mock_temporal
type ActivityFuture interface {
	Get(ctx workflow.Context, outRef any) error
	IsReady() bool
}

var errFutureGet = errors.New("future get failed")

func TestNewActivityFuture(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	var capturedFuture *temporal.ActivityFuture[string]

	env.RegisterWorkflow(func(ctx workflow.Context) error {
		mockFuture := mockTemporal.NewMockActivityFuture(ctrl)
		capturedFuture = temporal.NewActivityFuture[string]("test-future", mockFuture, "default-value")

		return nil
	})

	env.ExecuteWorkflow(func(ctx workflow.Context) error {
		mockFuture := mockTemporal.NewMockActivityFuture(ctrl)
		capturedFuture = temporal.NewActivityFuture[string]("test-future", mockFuture, "default-value")

		return nil
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.NotNil(t, capturedFuture)
	require.Equal(t, "test-future", capturedFuture.Name())
}

func TestActivityFuture_Name(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	var result string

	env.RegisterWorkflow(func(ctx workflow.Context) error {
		mockFuture := mockTemporal.NewMockActivityFuture(ctrl)
		activityFuture := temporal.NewActivityFuture[int]("my-activity-future", mockFuture, 42)
		result = activityFuture.Name()

		return nil
	})

	env.ExecuteWorkflow(func(ctx workflow.Context) error {
		mockFuture := mockTemporal.NewMockActivityFuture(ctrl)
		activityFuture := temporal.NewActivityFuture[int]("my-activity-future", mockFuture, 42)
		result = activityFuture.Name()

		return nil
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, "my-activity-future", result)
}

func TestActivityFuture_Future(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	mockFuture := mockTemporal.NewMockActivityFuture(ctrl)

	var retrievedFuture workflow.Future

	env.RegisterWorkflow(func(ctx workflow.Context) error {
		activityFuture := temporal.NewActivityFuture[string]("test", mockFuture, "")
		retrievedFuture = activityFuture.Future()

		return nil
	})

	env.ExecuteWorkflow(func(ctx workflow.Context) error {
		activityFuture := temporal.NewActivityFuture[string]("test", mockFuture, "")
		retrievedFuture = activityFuture.Future()

		return nil
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, mockFuture, retrievedFuture)
}

func TestActivityFuture_Get_Success(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	mockFuture := mockTemporal.NewMockActivityFuture(ctrl)
	mockFuture.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx workflow.Context, ptr *string) error {
		*ptr = "test-result"

		return nil
	})

	var result string

	env.RegisterWorkflow(func(ctx workflow.Context) error {
		activityFuture := temporal.NewActivityFuture[string]("test", mockFuture, "default")

		return activityFuture.Get(ctx, &result)
	})

	env.ExecuteWorkflow(func(ctx workflow.Context) error {
		activityFuture := temporal.NewActivityFuture[string]("test", mockFuture, "default")

		return activityFuture.Get(ctx, &result)
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, "test-result", result)
}

func TestActivityFuture_Get_Error(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	mockFuture := mockTemporal.NewMockActivityFuture(ctrl)
	mockFuture.EXPECT().Get(gomock.Any(), gomock.Any()).Return(errFutureGet)

	env.RegisterWorkflow(func(ctx workflow.Context) error {
		var result string

		activityFuture := temporal.NewActivityFuture[string]("test", mockFuture, "default")

		return activityFuture.Get(ctx, &result)
	})

	env.ExecuteWorkflow(func(ctx workflow.Context) error {
		var result string

		activityFuture := temporal.NewActivityFuture[string]("test", mockFuture, "default")

		return activityFuture.Get(ctx, &result)
	})

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
}

func TestActivityFuture_DifferentTypes(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	testSuite := &testsuite.WorkflowTestSuite{}

	tests := []struct {
		name         string
		defaultValue any
	}{
		{"string", "default-string"},
		{"int", 42},
		{"bool", true},
		{"float64", 3.14},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockFuture := mockTemporal.NewMockActivityFuture(ctrl)

			env := testSuite.NewTestWorkflowEnvironment()

			env.RegisterWorkflow(func(ctx workflow.Context) error {
				switch val := testCase.defaultValue.(type) {
				case string:
					future := temporal.NewActivityFuture[string]("test", mockFuture, val)
					require.NotNil(t, future)
				case int:
					future := temporal.NewActivityFuture[int]("test", mockFuture, val)
					require.NotNil(t, future)
				case bool:
					future := temporal.NewActivityFuture[bool]("test", mockFuture, val)
					require.NotNil(t, future)
				case float64:
					future := temporal.NewActivityFuture[float64]("test", mockFuture, val)
					require.NotNil(t, future)
				}

				return nil
			})

			env.ExecuteWorkflow(func(ctx workflow.Context) error {
				switch val := testCase.defaultValue.(type) {
				case string:
					future := temporal.NewActivityFuture[string]("test", mockFuture, val)
					require.NotNil(t, future)
				case int:
					future := temporal.NewActivityFuture[int]("test", mockFuture, val)
					require.NotNil(t, future)
				case bool:
					future := temporal.NewActivityFuture[bool]("test", mockFuture, val)
					require.NotNil(t, future)
				case float64:
					future := temporal.NewActivityFuture[float64]("test", mockFuture, val)
					require.NotNil(t, future)
				}

				return nil
			})

			require.True(t, env.IsWorkflowCompleted())
			require.NoError(t, env.GetWorkflowError())
		})
	}
}

func TestActivityFuture_ImplementsAwaitable(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	mockFuture := mockTemporal.NewMockActivityFuture(ctrl)
	mockFuture.EXPECT().IsReady().Return(true).AnyTimes()

	var awaitable temporal.Awaitable

	env.RegisterWorkflow(func(ctx workflow.Context) error {
		activityFuture := temporal.NewActivityFuture[string]("test", mockFuture, "default")
		awaitable = activityFuture

		return nil
	})

	env.ExecuteWorkflow(func(ctx workflow.Context) error {
		activityFuture := temporal.NewActivityFuture[string]("test", mockFuture, "default")
		awaitable = activityFuture

		return nil
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.NotNil(t, awaitable)
	require.Equal(t, "test", awaitable.Name())
	require.Equal(t, mockFuture, awaitable.Future())
}

func TestActivityFuture_ComplexTypes(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type CustomStruct struct {
		ID   int
		Name string
	}

	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	defaultValue := CustomStruct{ID: 1, Name: "default"}
	mockFuture := mockTemporal.NewMockActivityFuture(ctrl)

	var future *temporal.ActivityFuture[CustomStruct]

	env.RegisterWorkflow(func(ctx workflow.Context) error {
		future = temporal.NewActivityFuture[CustomStruct]("test", mockFuture, defaultValue)

		return nil
	})

	env.ExecuteWorkflow(func(ctx workflow.Context) error {
		future = temporal.NewActivityFuture[CustomStruct]("test", mockFuture, defaultValue)

		return nil
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.NotNil(t, future)
	require.Equal(t, "test", future.Name())
}

func TestActivityFuture_GetWithComplexType(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type Result struct {
		Value   string
		Success bool
	}

	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	expectedResult := Result{Value: "test-value", Success: true}
	mockFuture := mockTemporal.NewMockActivityFuture(ctrl)
	mockFuture.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx workflow.Context, ptr *Result) error {
		*ptr = expectedResult

		return nil
	})

	var actualResult Result

	env.RegisterWorkflow(func(ctx workflow.Context) error {
		future := temporal.NewActivityFuture[Result]("test", mockFuture, Result{})

		return future.Get(ctx, &actualResult)
	})

	env.ExecuteWorkflow(func(ctx workflow.Context) error {
		future := temporal.NewActivityFuture[Result]("test", mockFuture, Result{})

		return future.Get(ctx, &actualResult)
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, expectedResult, actualResult)
}
