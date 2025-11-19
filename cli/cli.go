package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/pixality-inc/golang-core/logger"
	"github.com/pixality-inc/golang-core/timetrack"
)

var (
	ErrExec     = errors.New("exec command")
	ErrExitCode = errors.New("exit code")
)

type Cli interface {
	Path() string

	RunCommand(
		ctx context.Context,
		env map[string]string,
		args ...string,
	) ([]byte, []byte, error)
}

type Impl struct {
	log      logger.Loggable
	toolPath string
}

func New(
	log logger.Loggable,
	toolPath string,
) *Impl {
	return &Impl{
		log:      log,
		toolPath: toolPath,
	}
}

func (c *Impl) Path() string {
	return c.toolPath
}

func (c *Impl) RunCommand(ctx context.Context, env map[string]string, args ...string) ([]byte, []byte, error) {
	cmdTimeTracker := timetrack.New()

	log := c.log.GetLogger(ctx)

	baseLogger := func(isSuccess bool, exitCode int, stdout []byte, stderr []byte) logger.Logger {
		cmdTimeTracker.Finish()

		fields := map[string]any{
			"logger":         "cmd",
			"exit_code":      exitCode,
			"success":        isSuccess,
			"args_count":     len(args),
			"execution_time": cmdTimeTracker.Duration().Milliseconds(),
		}

		if len(stderr) > 0 {
			fields["stderr"] = string(stderr)
			fields["stderr_len"] = len(stderr)
		}

		if len(stdout) > 0 {
			fields["stdout_len"] = len(stdout)
		}

		return log.WithFields(fields)
	}

	cmd := c.buildCommand(ctx, env, args...)

	command := cmd.String()

	exitCode, stdout, stderr, err := ExecCommand(cmd, true)
	switch {
	case err != nil:
		if len(stderr) > 0 {
			err = fmt.Errorf("%w: %s", errors.Join(ErrExec, err), stderr)
		} else {
			err = errors.Join(ErrExec, err)
		}
	case exitCode != 0:
		if len(stderr) > 0 {
			err = fmt.Errorf("%w: %d: %s", ErrExitCode, exitCode, stderr)
		} else {
			err = fmt.Errorf("%w: %d", ErrExitCode, exitCode)
		}
	default:
	}

	if err != nil || exitCode != 0 {
		baseLogger(false, exitCode, stdout, stderr).WithError(err).Error(ctx, command)

		return stdout, stderr, err
	}

	baseLogger(true, exitCode, stdout, stderr).Debug(command)

	return stdout, stderr, nil
}

func (c *Impl) buildCommand(ctx context.Context, env map[string]string, args ...string) *exec.Cmd {
	// #nosec G204
	cmd := exec.CommandContext(
		ctx,
		c.toolPath,
		args...,
	)

	if env != nil {
		var envs []string

		for k, v := range env {
			envs = append(envs, fmt.Sprintf("%s=%s", k, v))
		}

		cmd.Env = append(os.Environ(), envs...)
	}

	return cmd
}
