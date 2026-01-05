package control_flow_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/pixality-inc/golang-core/control_flow"
)

// nolint:paralleltest
func TestWaitForInterrupt_CancelsContext_OnInterrupt(t *testing.T) {
	controlFlow := control_flow.NewControlFlow(t.Context())

	done := make(chan struct{})

	go func() {
		controlFlow.WaitForInterrupt()
		close(done)
	}()

	// give WaitForInterrupt time to subscribe to the signal
	time.Sleep(50 * time.Millisecond)

	p, err := os.FindProcess(os.Getpid())
	require.NoError(t, err)

	err = p.Signal(os.Interrupt)
	require.NoError(t, err)

	select {
	case <-controlFlow.Context().Done():
	case <-time.After(2 * time.Second):
		require.Fail(t, "context was not canceled after interrupt")
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		require.Fail(t, "WaitForInterrupt did not return")
	}
}
