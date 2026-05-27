package logger

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	pkgerrors "github.com/pkg/errors"
)

var errBoom = errors.New("boom")

// jsonLogger builds a json logger writing into buf
func jsonLogger(stacktraceErrors bool, buf *bytes.Buffer) Logger {
	cfg := NewConfig(DebugLevel, JsonFormat, false, false, false, stacktraceErrors)

	return New(cfg).WithOutput(buf)
}

// TestWithErrorAttachesErrorField guards the regression where WithError dropped
// the error field when stacktrace_errors was enabled and the error carried no
// pkg/errors stack trace (zerolog Context.Err returns early on a nil stack)
func TestWithErrorAttachesErrorField(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name             string
		err              error
		stacktraceErrors bool
		wantStack        bool
	}{
		{"plain error, stacktrace_errors off", errBoom, false, false},
		{"plain error, stacktrace_errors on", errBoom, true, false},
		{"pkg/errors error, stacktrace_errors on", pkgerrors.WithStack(errBoom), true, true},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			jsonLogger(testCase.stacktraceErrors, &buf).WithError(testCase.err).Error("Request failed")

			out := buf.String()
			if !strings.Contains(out, `"error":"boom"`) {
				t.Fatalf("error field missing in output: %s", out)
			}

			hasStack := strings.Contains(out, `"stack":`)
			if hasStack != testCase.wantStack {
				t.Fatalf("stack field presence = %v, want %v: %s", hasStack, testCase.wantStack, out)
			}
		})
	}
}
