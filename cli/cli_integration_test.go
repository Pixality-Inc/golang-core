package cli_test

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pixality-inc/golang-core/cli"
	"github.com/pixality-inc/golang-core/logger"
)

func TestImpl_Path(t *testing.T) {
	t.Parallel()

	log := logger.NewLoggableImpl(nil)

	cliTest := cli.New(log, "/usr/bin/test")

	require.Equal(t, "/usr/bin/test", cliTest.Path())
}

func TestImpl_Exec_Success(t *testing.T) {
	t.Parallel()

	log := logger.NewLoggableImpl(nil)

	cliTest := cli.New(log, "/bin/echo")

	res, err := cliTest.Exec(t.Context(), []string{"hello"})
	require.NoError(t, err)

	require.Equal(t, 0, res.ExitCode())
	require.Equal(t, "hello\n", string(res.Stdout()))
	require.Empty(t, res.Stderr())
}

func TestImpl_Exec_ExitCodeError(t *testing.T) {
	t.Parallel()

	log := logger.NewLoggableImpl(nil)

	cliTest := cli.New(log, "/bin/sh")

	res, err := cliTest.Exec(t.Context(), []string{"-c", "exit 2"})
	require.Error(t, err)

	require.ErrorIs(t, err, cli.ErrExitCode)
	require.Equal(t, 2, res.ExitCode())
}

func TestImpl_Exec_ExitCodeError_WithStderr(t *testing.T) {
	t.Parallel()

	log := logger.NewLoggableImpl(nil)

	cliTest := cli.New(log, "/bin/sh")

	res, err := cliTest.Exec(
		t.Context(),
		[]string{"-c", "echo boom >&2; exit 3"},
	)

	require.Error(t, err)
	require.ErrorIs(t, err, cli.ErrExitCode)

	require.Equal(t, 3, res.ExitCode())
	require.Equal(t, "boom\n", string(res.Stderr()))
}

func TestImpl_Exec_ExecError(t *testing.T) {
	t.Parallel()

	log := logger.NewLoggableImpl(nil)

	cliTest := cli.New(log, "/definitely/not/exist")

	res, err := cliTest.Exec(t.Context(), nil)

	require.Error(t, err)
	require.ErrorIs(t, err, cli.ErrExec)
	require.NotErrorIs(t, err, cli.ErrExitCode)

	require.Equal(t, -1, res.ExitCode())
}

func TestImpl_Exec_ContextCancelled(t *testing.T) {
	t.Parallel()

	log := logger.NewLoggableImpl(nil)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	cliTest := cli.New(log, "/bin/echo")

	res, err := cliTest.Exec(ctx, []string{"hello"})

	require.Error(t, err)
	require.ErrorIs(t, err, cli.ErrExec)

	require.NotErrorIs(t, err, cli.ErrExitCode)
	require.NotEqual(t, 0, res.ExitCode())
}

func TestImpl_Exec_Stderr(t *testing.T) {
	t.Parallel()

	log := logger.NewLoggableImpl(nil)

	cliTest := cli.New(log, "/bin/sh")

	res, err := cliTest.Exec(
		t.Context(),
		[]string{"-c", "echo err >&2; exit 1"},
	)
	require.Error(t, err)

	require.Equal(t, "err\n", string(res.Stderr()))
}

func TestImpl_Exec_WorkDirAndEnv(t *testing.T) {
	t.Parallel()

	log := logger.NewLoggableImpl(nil)

	tmp := t.TempDir()

	cliTest := cli.New(log, "/bin/sh")

	res, err := cliTest.Exec(
		t.Context(),
		[]string{"-c", "pwd; echo $FOO"},
		cli.WithWorkDir(tmp),
		cli.WithEnv("FOO", "bar"),
	)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(res.Stdout())), "\n")
	require.Len(t, lines, 2)

	pwdFromCmd := lines[0]
	envValue := lines[1]

	expectedPath, err := filepath.EvalSymlinks(tmp)
	require.NoError(t, err)

	actualPath, err := filepath.EvalSymlinks(pwdFromCmd)
	require.NoError(t, err)

	require.Equal(t, expectedPath, actualPath)
	require.Equal(t, "bar", envValue)
}

func TestExecCommand(t *testing.T) {
	t.Parallel()

	cmd := exec.CommandContext(t.Context(), "/bin/echo", "ok")

	exitCode, stdout, stderr, err := cli.ExecCommand(cmd, true)
	require.NoError(t, err)

	require.Equal(t, 0, exitCode)
	require.Equal(t, "ok\n", string(stdout))
	require.Empty(t, stderr)
}

func TestImpl_Exec_AllOptionsUsed(t *testing.T) {
	t.Parallel()

	log := logger.NewLoggableImpl(nil)

	tmp := t.TempDir()

	var stdoutBuf, stderrBuf bytes.Buffer

	cliTest := cli.New(log, "/bin/sh")

	res, err := cliTest.Exec(
		t.Context(),
		[]string{
			"-c",
			`
				pwd
				echo $FOO
				echo $BAR
				echo "out"
				echo "err" >&2
				exit 1
			`,
		},
		cli.WithWorkDir(tmp),
		cli.WithStdout(&stdoutBuf),
		cli.WithStderr(&stderrBuf),
		cli.WithEnv("FOO", "foo-value"),
		cli.WithEnvs(map[string]string{
			"BAR": "bar-value",
		}),
	)

	require.Error(t, err)
	require.ErrorIs(t, err, cli.ErrExitCode)

	require.Contains(t, string(res.Stdout()), "out")
	require.Contains(t, string(res.Stderr()), "err")

	lines := strings.Split(strings.TrimSpace(string(res.Stdout())), "\n")

	expectedDir, err := filepath.EvalSymlinks(tmp)
	require.NoError(t, err)

	actualDir, err := filepath.EvalSymlinks(lines[0])
	require.NoError(t, err)

	require.Equal(t, expectedDir, actualDir)

	require.Contains(t, lines, "foo-value")
	require.Contains(t, lines, "bar-value")
}
