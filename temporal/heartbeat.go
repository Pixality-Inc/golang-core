package temporal

import (
	"context"
	"sync"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/interceptor"
)

// heartbeatsPerTimeout is how many heartbeats a running activity sends within its heartbeat timeout,
// so that one heartbeat lost or late on the way is not taken for a lost worker
const heartbeatsPerTimeout = 4

// heartbeatInterceptor heartbeats the activities registered with a heartbeat timeout for as long as
// they run. the timeout is how temporal notices a worker that is gone (killed, redeployed) long
// before StartToClose, and an activity that waits out a storage outage for hours must not be taken
// for a lost one. heartbeating here rather than in the activity code keeps the two from drifting
// apart: a timeout set in config without a heartbeat in code would fail every attempt.
//
// the activity config of the worker opts in (an activity that heartbeats on its own progress keeps
// doing so untouched), while the interval follows the timeout the activity was scheduled with, so a
// scheduler whose config differs from the worker's still gets heartbeats in time, and none when it
// set no timeout. the heartbeat tells that the worker is alive, not that the activity makes
// progress, so a call hung inside it is still bounded by StartToClose only. it carries no details,
// so it does not suit an activity that keeps its progress in heartbeat details
type heartbeatInterceptor struct {
	interceptor.WorkerInterceptorBase

	// activity name -> struct{}, written on registration and read by every execution
	activities sync.Map
}

func (i *heartbeatInterceptor) register(name ActivityName) {
	i.activities.Store(string(name), struct{}{})
}

func (i *heartbeatInterceptor) InterceptActivity(
	_ context.Context,
	next interceptor.ActivityInboundInterceptor,
) interceptor.ActivityInboundInterceptor {
	return &heartbeatActivityInterceptor{
		ActivityInboundInterceptorBase: interceptor.ActivityInboundInterceptorBase{Next: next},
		root:                           i,
	}
}

type heartbeatActivityInterceptor struct {
	interceptor.ActivityInboundInterceptorBase

	root *heartbeatInterceptor
}

func (a *heartbeatActivityInterceptor) ExecuteActivity(
	ctx context.Context,
	in *interceptor.ExecuteActivityInput,
) (any, error) {
	info := activity.GetInfo(ctx)

	if _, ok := a.root.activities.Load(info.ActivityType.Name); ok && info.HeartbeatTimeout > 0 {
		stop := startHeartbeat(ctx, max(info.HeartbeatTimeout/heartbeatsPerTimeout, 1))
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
