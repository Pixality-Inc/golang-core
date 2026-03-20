package cli

import (
	"bytes"
	"errors"
	"io"
	"os/exec"
	"syscall"
)

// ExecCommand exitCode, stdout, stderr, error
func ExecCommand(cmd *exec.Cmd, failIfExitCodeNotZero bool) (int, []byte, []byte, error) {
	stdoutBuffer := bytes.NewBuffer(nil)

	if cmd.Stdout == nil {
		cmd.Stdout = stdoutBuffer
	} else {
		cmd.Stdout = io.MultiWriter(stdoutBuffer, cmd.Stdout)
	}

	stderrBuffer := bytes.NewBuffer(nil)

	if cmd.Stderr == nil {
		cmd.Stderr = stderrBuffer
	} else {
		cmd.Stderr = io.MultiWriter(cmd.Stderr, stderrBuffer)
	}

	exitCode := 0

	if err := cmd.Run(); err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			return -1, stdoutBuffer.Bytes(), stderrBuffer.Bytes(), err
		}

		status, ok := exitError.Sys().(syscall.WaitStatus)
		if !ok {
			return -1, stdoutBuffer.Bytes(), stderrBuffer.Bytes(), err
		}

		exitCode = status.ExitStatus()
		if failIfExitCodeNotZero && exitCode != 0 {
			return exitCode, stdoutBuffer.Bytes(), stderrBuffer.Bytes(), err
		}
	}

	return exitCode, stdoutBuffer.Bytes(), stderrBuffer.Bytes(), nil
}
