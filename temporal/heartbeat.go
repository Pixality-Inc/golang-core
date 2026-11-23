package temporal

import (
	"context"
	"sync"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/interceptor"
)

const (
	// heartbeatsPerTimeout is how often within its heartbeat timeout the worker records a heartbeat
	// of a running activity. the sdk sends one at most every 0.8 of the timeout and at most every
	// minute, so for a timeout of minutes every record is sent and a lost one is not taken for a
	// lost worker
	heartbeatsPerTimeout = 4

	// minHeartbeatInterval keeps a mistyped timeout (1us) from turning the ticker into a busy loop.
	// the sdk throttles the sends anyway
	minHeartbeatInterval = 100 * time.Millisecond
)

// heartbeatWorkerInterceptor heartbeats a running activity when both the worker registered it
// with a heartbeat timeout in its config and the activity was scheduled with one; the interval
// follows the scheduled timeout. a scheduler that sets the timeout for an activity whose worker
// neither opts in nor heartbeats itself fails every attempt longer than the timeout.
//
// the heartbeat tells that the worker is alive, not that the activity makes progress: a call hung
// inside it is bounded by StartToClose only. it carries no details and overwrites the ones the
// activity recorded itself. through the heartbeat the sdk also cancels the activity: on a
// cancellation, on a timed out attempt, and on a temporal frontend failing the heartbeat calls for
// about the throttle interval
type heartbeatWorkerInterceptor struct {
	interceptor.WorkerInterceptorBase

	// activity name -> struct{}, written on registration and read by every execution
	activities sync.Map
}

func (i *heartbeatWorkerInterceptor) register(name ActivityName) {
	i.activities.Store(string(name), struct{}{})
}

func (i *heartbeatWorkerInterceptor) InterceptActivity(
	_ context.Context,
	next interceptor.ActivityInboundInterceptor,
) interceptor.ActivityInboundInterceptor {
	return &heartbeatActivityInboundInterceptor{
		ActivityInboundInterceptorBase: interceptor.ActivityInboundInterceptorBase{Next: next},
		root:                           i,
	}
}

type heartbeatActivityInboundInterceptor struct {
	interceptor.ActivityInboundInterceptorBase

	root *heartbeatWorkerInterceptor
}

func (a *heartbeatActivityInboundInterceptor) ExecuteActivity(
	ctx context.Context,
	in *interceptor.ExecuteActivityInput,
) (any, error) {
	info := activity.GetInfo(ctx)

	if _, ok := a.root.activities.Load(info.ActivityType.Name); ok && info.HeartbeatTimeout > 0 {
		stop := startHeartbeat(ctx, max(info.HeartbeatTimeout/heartbeatsPerTimeout, minHeartbeatInterval))
		defer stop()
	}

	return a.Next.ExecuteActivity(ctx, in)
}

// startHeartbeat heartbeats the activity of ctx every interval until the returned stop is called,
// which returns once the heartbeat has stopped
func startHeartbeat(ctx context.Context, interval time.Duration) func() {
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})

	go func() {
		defer close(done)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				activity.RecordHeartbeat(ctx)
			}
		}
	}()

	return func() {
		cancel()
		<-done
	}
}
