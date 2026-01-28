package temporal_test

import (
	"testing"
	"time"

	"github.com/pixality-inc/golang-core/temporal"
	"github.com/stretchr/testify/require"
)

func TestActivityConfig_Fields(t *testing.T) {
	t.Parallel()

	config := temporal.ActivityConfig{
		Name:                    "test-activity",
		Queue:                   "test-queue",
		Timeout:                 30 * time.Second,
		MaxAttempts:             5,
		RetryInitialInterval:    2 * time.Second,
		RetryBackoffCoefficient: 1.5,
		RetryMaximumInterval:    5 * time.Minute,
	}

	require.Equal(t, temporal.ActivityName("test-activity"), config.Name)
	require.Equal(t, temporal.QueueName("test-queue"), config.Queue)
	require.Equal(t, 30*time.Second, config.Timeout)
	require.Equal(t, 5, config.MaxAttempts)
	require.Equal(t, 2*time.Second, config.RetryInitialInterval)
	require.InEpsilon(t, 1.5, config.RetryBackoffCoefficient, 0.0)
	require.Equal(t, 5*time.Minute, config.RetryMaximumInterval)
}
