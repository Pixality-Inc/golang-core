package tests

import (
	"context"
	"os"
	"path"
	"runtime"
	"syscall"
	"testing"

	"github.com/pixality-inc/golang-core/flow"
	mockFlow "github.com/pixality-inc/golang-core/flow/mocks"
	"github.com/pixality-inc/golang-core/storage"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	testCommandToRun       = "true"
	testFailedCommandToRun = "false"
	testEchoCommandToRun   = "echo"
	testMkdirCommandToRun  = "mkdir"
)

//nolint:maintidx
func TestFlow(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	mockCtrl := gomock.NewController(t)

	noTemplateDriverMock := func() flow.TemplateDriver {
		return nil
	}

	noScriptDriverMock := func() flow.ScriptDriver {
		return nil
	}

	noLocalStorageMock := func() storage.LocalStorage {
		return nil
	}

	newDefaultEnv := func() *flow.Env {
		return flow.NewEnv("", make(map[string]any))
	}

	newEnvWithWorkDir := func(workDir string) *flow.Env {
		return flow.NewEnv(workDir, make(map[string]any))
	}

	defaultEnv := newDefaultEnv()

	mkdirErrorMessage := "mkdir: cannot create directory ‘.’: File exists\n"
	if runtime.GOOS == "darwin" {
		mkdirErrorMessage = "mkdir: .: File exists\n"
	}

	testWorkDir := path.Join(os.TempDir(), "flow_test")
	if err := os.MkdirAll(testWorkDir, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	testsAnotherWorkDir := path.Join(os.TempDir(), "flow_test_another")
	if err := os.MkdirAll(testsAnotherWorkDir, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	testsWorkDirFile := path.Join(testWorkDir, "file")

	//nolint:gosec
	if err := os.WriteFile(testsWorkDirFile, nil, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if fErr := os.RemoveAll(testWorkDir); fErr != nil {
			t.Errorf("failed to remove tests work dir %s: %v", testWorkDir, fErr)
		}

		if fErr := os.RemoveAll(testsAnotherWorkDir); fErr != nil {
			t.Errorf("failed to remove tests another work dir %s: %v", testsAnotherWorkDir, fErr)
		}
	})

	tests := []struct {
		name               string
		actions            []flow.Action
		env                *flow.Env
		templateDriverMock func() flow.TemplateDriver
		scriptDriverMock   func() flow.ScriptDriver
		localStorageMock   func() storage.LocalStorage
		wantResult         *flow.Result
		wantResultLen      int
		wantErr            error
		after              func(t *testing.T)
	}{
		{
			name:               "empty",
			templateDriverMock: noTemplateDriverMock,
			scriptDriverMock:   noScriptDriverMock,
			localStorageMock:   noLocalStorageMock,
			wantResultLen:      0,
			wantErr:            nil,
		},
		{
			name: "one_action_no_options",
			actions: []flow.Action{
				flow.NewAction("test"),
			},
			templateDriverMock: noTemplateDriverMock,
			scriptDriverMock:   noScriptDriverMock,
			localStorageMock:   noLocalStorageMock,
			wantErr:            flow.ErrActionNoOptions,
		},
		{
			name: "too_many_options",
			actions: []flow.Action{
				flow.NewAction("test").
					WithCommand(testCommandToRun).
					WithScript("hello"),
			},
			templateDriverMock: noTemplateDriverMock,
			scriptDriverMock:   noScriptDriverMock,
			localStorageMock:   noLocalStorageMock,
			wantErr:            flow.ErrActionTooManyOptions,
		},
		{
			name: "too_many_options2",
			actions: []flow.Action{
				flow.NewAction("test").
					WithCommand(testCommandToRun).
					WithScriptFile("hello.js"),
			},
			templateDriverMock: noTemplateDriverMock,
			scriptDriverMock:   noScriptDriverMock,
			localStorageMock:   noLocalStorageMock,
			wantErr:            flow.ErrActionTooManyOptions,
		},
		{
			name: "too_many_options3",
			actions: []flow.Action{
				flow.NewAction("test").
					WithScript("hello").
					WithScriptFile("hello.js"),
			},
			templateDriverMock: noTemplateDriverMock,
			scriptDriverMock:   noScriptDriverMock,
			localStorageMock:   noLocalStorageMock,
			wantErr:            flow.ErrActionTooManyOptions,
		},
		{
			name: "args_too_many_options",
			actions: []flow.Action{
				flow.NewAction("test").
					WithCommand(testCommandToRun).
					WithArgs("1", "2", "3").
					WithArgsTemplate("hello"),
			},
			env:                defaultEnv,
			templateDriverMock: noTemplateDriverMock,
			scriptDriverMock:   noScriptDriverMock,
			localStorageMock:   noLocalStorageMock,
			wantErr:            flow.ErrActionArgsTooManyOptions,
		},
		{
			name: "args_too_many_options2",
			actions: []flow.Action{
				flow.NewAction("test").
					WithCommand(testCommandToRun).
					WithArgs("1", "2", "3").
					WithArgsScript("hello"),
			},
			env:                defaultEnv,
			templateDriverMock: noTemplateDriverMock,
			scriptDriverMock:   noScriptDriverMock,
			localStorageMock:   noLocalStorageMock,
			wantErr:            flow.ErrActionArgsTooManyOptions,
		},
		{
			name: "args_too_many_options3",
			actions: []flow.Action{
				flow.NewAction("test").
					WithCommand(testCommandToRun).
					WithArgsTemplate("hello").
					WithArgsScript("hello"),
			},
			env:                defaultEnv,
			templateDriverMock: noTemplateDriverMock,
			scriptDriverMock:   noScriptDriverMock,
			localStorageMock:   noLocalStorageMock,
			wantErr:            flow.ErrActionArgsTooManyOptions,
		},
		{
			name: "env_too_many_options",
			actions: []flow.Action{
				flow.NewAction("test").
					WithCommand(testCommandToRun).
					WithEnv("foo", "bar").
					WithEnvTemplate("hello"),
			},
			env:                defaultEnv,
			templateDriverMock: noTemplateDriverMock,
			scriptDriverMock:   noScriptDriverMock,
			localStorageMock:   noLocalStorageMock,
			wantErr:            flow.ErrActionEnvTooManyOptions,
		},
		{
			name: "env_too_many_options2",
			actions: []flow.Action{
				flow.NewAction("test").
					WithCommand(testCommandToRun).
					WithEnvs(map[string]string{
						"foo": "bar",
					}).
					WithEnvScript("hello"),
			},
			env:                defaultEnv,
			templateDriverMock: noTemplateDriverMock,
			scriptDriverMock:   noScriptDriverMock,
			localStorageMock:   noLocalStorageMock,
			wantErr:            flow.ErrActionEnvTooManyOptions,
		},
		{
			name: "env_too_many_options3",
			actions: []flow.Action{
				flow.NewAction("test").
					WithCommand(testCommandToRun).
					WithEnvTemplate("hello").
					WithEnvScript("hello"),
			},
			env:                defaultEnv,
			templateDriverMock: noTemplateDriverMock,
			scriptDriverMock:   noScriptDriverMock,
			localStorageMock:   noLocalStorageMock,
			wantErr:            flow.ErrActionEnvTooManyOptions,
		},
		{
			name: "env_work_dir_failed",
			actions: []flow.Action{
				flow.NewAction("test").
					WithCommand(testCommandToRun).
					WithWorkDir(testsWorkDirFile),
			},
			templateDriverMock: noTemplateDriverMock,
			scriptDriverMock:   noScriptDriverMock,
			localStorageMock:   noLocalStorageMock,
			wantErr:            syscall.ENOTDIR,
		},
		{
			name: "command_stdout",
			actions: []flow.Action{
				flow.NewAction("test").
					WithCommand(testEchoCommandToRun, "hello"),
			},
			env:                defaultEnv,
			templateDriverMock: noTemplateDriverMock,
			scriptDriverMock:   noScriptDriverMock,
			localStorageMock:   noLocalStorageMock,
			wantResult: &flow.Result{
				ActionsResponses: map[string]*flow.ActionResponse{
					"test": flow.NewActionResponse().WithStdout("hello\n"),
				},
			},
			wantResultLen: 1,
		},
		{
			name: "command_two_args_stdout",
			actions: []flow.Action{
				flow.NewAction("test").
					WithCommand(testEchoCommandToRun).
					WithArgs("-n", "hello"),
			},
			env:                defaultEnv,
			templateDriverMock: noTemplateDriverMock,
			scriptDriverMock:   noScriptDriverMock,
			localStorageMock:   noLocalStorageMock,
			wantResult: &flow.Result{
				ActionsResponses: map[string]*flow.ActionResponse{
					"test": flow.NewActionResponse().WithStdout("hello"),
				},
			},
			wantResultLen: 1,
		},
		{
			name: "command_fail",
			actions: []flow.Action{
				flow.NewAction("test").
					WithCommand(testFailedCommandToRun),
			},
			env:                defaultEnv,
			templateDriverMock: noTemplateDriverMock,
			scriptDriverMock:   noScriptDriverMock,
			localStorageMock:   noLocalStorageMock,
			wantResultLen:      1,
			wantErr:            flow.ErrCommandFailed,
		},
		{
			name: "command_fail_exit_code",
			actions: []flow.Action{
				flow.NewAction("test").
					WithCommand(testFailedCommandToRun).
					WithFailIfNonZeroCode(false),
			},
			env:                defaultEnv,
			templateDriverMock: noTemplateDriverMock,
			scriptDriverMock:   noScriptDriverMock,
			localStorageMock:   noLocalStorageMock,
			wantResult: &flow.Result{
				ActionsResponses: map[string]*flow.ActionResponse{
					"test": flow.NewActionResponse().WithExitCode(1),
				},
			},
			wantResultLen: 1,
		},
		{
			name: "command_stderr",
			actions: []flow.Action{
				flow.NewAction("test").
					WithCommand(testMkdirCommandToRun, ".").
					WithFailIfNonZeroCode(false),
			},
			env:                defaultEnv,
			templateDriverMock: noTemplateDriverMock,
			scriptDriverMock:   noScriptDriverMock,
			localStorageMock:   noLocalStorageMock,
			wantResult: &flow.Result{
				ActionsResponses: map[string]*flow.ActionResponse{
					"test": flow.NewActionResponse().WithExitCode(1).WithStderr(mkdirErrorMessage),
				},
			},
			wantResultLen: 1,
		},
		{
			name: "when_true",
			actions: []flow.Action{
				flow.NewAction("test").
					WithCommand(testCommandToRun).
					WithWhen("hello"),
			},
			env:                defaultEnv,
			templateDriverMock: noTemplateDriverMock,
			scriptDriverMock: func() flow.ScriptDriver {
				mock := mockFlow.NewMockScriptDriver(mockCtrl)

				mock.EXPECT().Execute(ctx, defaultEnv, "action.test.when", "hello").Return("value", nil)
				mock.EXPECT().ValueToBool("value").Return(true, nil)

				return mock
			},
			localStorageMock: noLocalStorageMock,
			wantResult: &flow.Result{
				ActionsResponses: map[string]*flow.ActionResponse{
					"test": flow.NewActionResponse(),
				},
			},
			wantResultLen: 1,
		},
		{
			name: "when_false",
			actions: []flow.Action{
				flow.NewAction("test").
					WithCommand(testCommandToRun).
					WithWhen("hello"),
			},
			env:                defaultEnv,
			templateDriverMock: noTemplateDriverMock,
			scriptDriverMock: func() flow.ScriptDriver {
				mock := mockFlow.NewMockScriptDriver(mockCtrl)

				mock.EXPECT().Execute(ctx, defaultEnv, "action.test.when", "hello").Return("value", nil)
				mock.EXPECT().ValueToBool("value").Return(false, nil)

				return mock
			},
			localStorageMock: noLocalStorageMock,
			wantResult: &flow.Result{
				ActionsResponses: map[string]*flow.ActionResponse{
					"test": flow.NewActionResponse().WithSkipped(true),
				},
			},
			wantResultLen: 1,
		},
		{
			name: "stdout_file",
			actions: []flow.Action{
				flow.NewAction("test").
					WithCommand(testEchoCommandToRun, "hello").
					WithStdout("stdout_file.txt"),
			},
			env:                newEnvWithWorkDir(testWorkDir),
			templateDriverMock: noTemplateDriverMock,
			scriptDriverMock:   noScriptDriverMock,
			localStorageMock:   noLocalStorageMock,
			wantResult: &flow.Result{
				ActionsResponses: map[string]*flow.ActionResponse{
					"test": flow.NewActionResponse(),
				},
			},
			wantResultLen: 1,
			after: func(t *testing.T) {
				t.Helper()

				buf, err := os.ReadFile(path.Join(testWorkDir, "stdout_file.txt"))
				require.NoError(t, err)
				require.Equal(t, "hello\n", string(buf))
			},
		},
		{
			name: "stderr_file",
			actions: []flow.Action{
				flow.NewAction("test").
					WithCommand(testMkdirCommandToRun, ".").
					WithFailIfNonZeroCode(false).
					WithStderr("stdout_file.error.txt"),
			},
			env:                newEnvWithWorkDir(testWorkDir),
			templateDriverMock: noTemplateDriverMock,
			scriptDriverMock:   noScriptDriverMock,
			localStorageMock:   noLocalStorageMock,
			wantResult: &flow.Result{
				ActionsResponses: map[string]*flow.ActionResponse{
					"test": flow.NewActionResponse().WithExitCode(1),
				},
			},
			wantResultLen: 1,
			after: func(t *testing.T) {
				t.Helper()

				buf, err := os.ReadFile(path.Join(testWorkDir, "stdout_file.error.txt"))
				require.NoError(t, err)
				require.Equal(t, mkdirErrorMessage, string(buf))
			},
		},
		{
			name: "stdout_file_work_dir",
			actions: []flow.Action{
				flow.NewAction("test").
					WithCommand(testEchoCommandToRun, "hello").
					WithWorkDir(testsAnotherWorkDir).
					WithStdout("stdout_file.txt"),
			},
			env:                newEnvWithWorkDir(testWorkDir),
			templateDriverMock: noTemplateDriverMock,
			scriptDriverMock:   noScriptDriverMock,
			localStorageMock:   noLocalStorageMock,
			wantResult: &flow.Result{
				ActionsResponses: map[string]*flow.ActionResponse{
					"test": flow.NewActionResponse(),
				},
			},
			wantResultLen: 1,
			after: func(t *testing.T) {
				t.Helper()

				buf, err := os.ReadFile(path.Join(testsAnotherWorkDir, "stdout_file.txt"))
				require.NoError(t, err)
				require.Equal(t, "hello\n", string(buf))
			},
		},
		{
			name: "stderr_file_work_dir",
			actions: []flow.Action{
				flow.NewAction("test").
					WithCommand(testMkdirCommandToRun, ".").
					WithWorkDir(testsAnotherWorkDir).
					WithFailIfNonZeroCode(false).
					WithStderr("stdout_file.error.txt"),
			},
			env:                newEnvWithWorkDir(testWorkDir),
			templateDriverMock: noTemplateDriverMock,
			scriptDriverMock:   noScriptDriverMock,
			localStorageMock:   noLocalStorageMock,
			wantResult: &flow.Result{
				ActionsResponses: map[string]*flow.ActionResponse{
					"test": flow.NewActionResponse().WithExitCode(1),
				},
			},
			wantResultLen: 1,
			after: func(t *testing.T) {
				t.Helper()

				buf, err := os.ReadFile(path.Join(testsAnotherWorkDir, "stdout_file.error.txt"))
				require.NoError(t, err)
				require.Equal(t, mkdirErrorMessage, string(buf))
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			flowEngine := flow.New(
				&flow.Config{
					Actions: testCase.actions,
				},
				testCase.localStorageMock(),
				testCase.templateDriverMock(),
				testCase.scriptDriverMock(),
			)

			result, err := flowEngine.Run(ctx, testCase.env)

			if testCase.wantErr != nil {
				require.ErrorIs(t, err, testCase.wantErr)
			} else {
				require.NoError(t, err)
			}

			if testCase.wantResult != nil {
				require.NotNil(t, result)
				require.Len(t, result.ActionsResponses, len(testCase.wantResult.ActionsResponses))

				for index, response := range testCase.wantResult.ActionsResponses {
					resultResponse, ok := result.ActionsResponses[index]
					require.True(t, ok, "index %s", index)

					require.Equal(t, response.ErrorCode, resultResponse.ErrorCode, "index %s", index)
					require.Equal(t, response.Stdout, resultResponse.Stdout, "index %s", index)
					require.Equal(t, response.Stderr, resultResponse.Stderr, "index %s", index)
					require.Equal(t, response.Skipped, resultResponse.Skipped, "index %s", index)
					require.Equal(t, response.Result, resultResponse.Result, "index %s", index)
				}
			}

			if testCase.wantErr != nil {
				require.Nil(t, result)
			}

			if testCase.wantErr == nil && testCase.wantResultLen > -1 {
				require.NotNil(t, result)
				require.Len(t, result.ActionsResponses, testCase.wantResultLen)
			}

			if testCase.after != nil {
				testCase.after(t)
			}
		})
	}
}
