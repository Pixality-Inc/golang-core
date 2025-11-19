package cli

import (
	"bufio"
	"bytes"
	"errors"
	"os/exec"
	"syscall"
)

// ExecCommand exitCode, stdout, stderr, error
func ExecCommand(cmd *exec.Cmd, failIfExitCodeNotZero bool) (int, []byte, []byte, error) {
	var stdoutBuffer bytes.Buffer

	stdoutWriter := bufio.NewWriter(&stdoutBuffer)

	cmd.Stdout = stdoutWriter

	var stderrBuffer bytes.Buffer

	stderrWriter := bufio.NewWriter(&stderrBuffer)

	cmd.Stderr = stderrWriter

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
