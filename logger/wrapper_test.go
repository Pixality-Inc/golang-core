package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestWrapper() *Wrapper {
	return NewWrapper(NewLoggableImplWithService("test").GetLoggerWithoutContext())
}

func TestWrapperWithArgsEvenPairs(t *testing.T) {
	t.Parallel()

	entry, args := newTestWrapper().withArgs("message", "key", "value", "other", 42)

	require.NotNil(t, entry)
	assert.Equal(t, []any{"message"}, args)
}

func TestWrapperWithArgsNoKeyvals(t *testing.T) {
	t.Parallel()

	_, args := newTestWrapper().withArgs("message")

	assert.Equal(t, []any{"message"}, args)
}

func TestWrapperWithArgsOddCount(t *testing.T) {
	t.Parallel()

	_, args := newTestWrapper().withArgs("message", "dangling")

	assert.Equal(t, []any{"message", " ", "dangling"}, args)
}

func TestWrapperWithArgsNonStringKey(t *testing.T) {
	t.Parallel()

	_, args := newTestWrapper().withArgs("message", 42, "value")

	assert.Equal(t, []any{"message", " ", 42, " ", "value"}, args)
}

func TestWrapperLogMethods(t *testing.T) {
	t.Parallel()

	wrapper := newTestWrapper()

	wrapper.Debug("debug message", "key", "value")
	wrapper.Info("info message")
	wrapper.Warn("warn message", "dangling")
	wrapper.Error("error message", 42, "value")
}

func TestLoggableImplNilFields(t *testing.T) {
	t.Parallel()

	loggable := NewLoggableImpl(nil)

	require.NotNil(t, loggable.GetLoggerWithoutContext())
	require.NotNil(t, loggable.GetLogger(t.Context()))
}

func TestLoggableImplWithServiceAndFields(t *testing.T) {
	t.Parallel()

	withNil := NewLoggableImplWithServiceAndFields("svc", nil)
	require.NotNil(t, withNil.GetLoggerWithoutContext())

	withFields := NewLoggableImplWithServiceAndFields("svc", Fields{"component": "worker"})
	require.NotNil(t, withFields.GetLoggerWithoutContext())
	assert.Len(t, withFields.extraFields, 2)
	assert.Equal(t, "worker", withFields.extraFields["component"])
}

func TestConfigGetters(t *testing.T) {
	t.Parallel()

	config := NewConfig(DebugLevel, JsonFormat, true, false, true, false)

	assert.Equal(t, DebugLevel, config.Level())
	assert.Equal(t, JsonFormat, config.Format())
	assert.True(t, config.WithTimestamp())
	assert.False(t, config.WithColors())
	assert.True(t, config.WithStacktrace())
	assert.False(t, config.WithStacktraceErrors())
}

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	assert.Equal(t, InfoLevel, DefaultConfig.Level())
	assert.Equal(t, TextFormat, DefaultConfig.Format())
	assert.True(t, DefaultConfig.WithTimestamp())
	assert.True(t, DefaultConfig.WithColors())
	assert.False(t, DefaultConfig.WithStacktrace())
	assert.False(t, DefaultConfig.WithStacktraceErrors())
}
