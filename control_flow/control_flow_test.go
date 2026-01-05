package control_flow_test

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/pixality-inc/golang-core/control_flow"
)

type fakeShutdown struct {
	mu    sync.Mutex
	calls int
	err   error
}

func (f *fakeShutdown) Stop() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls++

	return f.err
}

func (f *fakeShutdown) Calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.calls
}

type fakeShutdownWithName struct {
	fakeShutdown

	name string
}

func (f *fakeShutdownWithName) Name() string {
	return f.name
}

type fakeClosable struct {
	calls int
}

func (f *fakeClosable) Close() {
	f.calls++
}

type fakeStoppable struct {
	calls int
}

func (f *fakeStoppable) Stop() {
	f.calls++
}

func TestNewControlFlow(t *testing.T) {
	t.Parallel()

	controlFlow := control_flow.NewControlFlow(t.Context())

	require.NotNil(t, controlFlow)
	require.NotNil(t, controlFlow.Context())
	require.NotNil(t, controlFlow.Cancel())
}

func TestControlFlow_Cancel(t *testing.T) {
	t.Parallel()

	controlFlow := control_flow.NewControlFlow(t.Context())

	cancel := controlFlow.Cancel()
	cancel()

	select {
	case <-controlFlow.Context().Done():
	case <-time.After(time.Second):
		require.Fail(t, "context didn't cancel")
	}
}

func TestRegisterShutdownService_AndShutdown(t *testing.T) {
	t.Parallel()

	controlFlow := control_flow.NewControlFlow(t.Context())

	svc := &fakeShutdown{}

	controlFlow.RegisterShutdownService("svc", svc)
	controlFlow.Shutdown()

	require.Equal(t, 1, svc.Calls())
}

func TestRegisterShutdownServiceWithName(t *testing.T) {
	t.Parallel()

	controlFlow := control_flow.NewControlFlow(t.Context())

	svc := &fakeShutdownWithName{
		name: "named-service",
	}

	controlFlow.RegisterShutdownServiceWithName(svc)
	controlFlow.Shutdown()

	require.Equal(t, 1, svc.Calls())
}

func TestRegisterClosableService(t *testing.T) {
	t.Parallel()

	controlFlow := control_flow.NewControlFlow(t.Context())

	closable := &fakeClosable{}

	controlFlow.RegisterClosableService("closable", closable)
	controlFlow.Shutdown()

	require.Equal(t, 1, closable.calls)
}

func TestRegisterStoppableService(t *testing.T) {
	t.Parallel()

	controlFlow := control_flow.NewControlFlow(t.Context())

	stoppable := &fakeStoppable{}

	controlFlow.RegisterStoppableService("stoppable", stoppable)
	controlFlow.Shutdown()

	require.Equal(t, 1, stoppable.calls)
}

func TestShutdown_MultipleServices(t *testing.T) {
	t.Parallel()

	controlFlow := control_flow.NewControlFlow(t.Context())

	shutdown1 := &fakeShutdown{}
	shutdown2 := &fakeShutdown{}

	controlFlow.RegisterShutdownService("shutdown1", shutdown1)
	controlFlow.RegisterShutdownService("shutdown2", shutdown2)

	controlFlow.Shutdown()

	require.Equal(t, 1, shutdown1.Calls())
	require.Equal(t, 1, shutdown2.Calls())
}
