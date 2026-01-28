package temporal_test

import (
	"testing"
	"time"

	"github.com/pixality-inc/golang-core/temporal"
	"github.com/stretchr/testify/require"
)

func TestActivity(t *testing.T) {
	t.Parallel()

	config := temporal.ActivityConfig{
		Name:                    "my-activity",
		Queue:                   "my-queue",
		Timeout:                 time.Minute,
		MaxAttempts:             3,
		RetryInitialInterval:    time.Second,
		RetryBackoffCoefficient: 2.0,
		RetryMaximumInterval:    time.Minute,
	}

	activity := temporal.NewActivityImpl(nil, config)

	require.Equal(t, temporal.ActivityName("my-activity"), activity.Name())
	require.Equal(t, temporal.QueueName("my-queue"), activity.Queue())
	require.Equal(t, time.Minute, activity.Timeout())
	require.Equal(t, 3, activity.MaxAttempts())
	require.Equal(t, time.Second, activity.RetryInitialInterval())
	require.InEpsilon(t, 2.0, activity.RetryBackoffCoefficient(), 0.0)
	require.Equal(t, time.Minute, activity.RetryMaximumInterval())
	require.NotNil(t, activity.GetLogger(t.Context()))
	require.NotNil(t, activity.GetLoggerWithoutContext())
}
