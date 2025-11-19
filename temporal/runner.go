package temporal

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var (
	ErrWaitForChannels  = errors.New("WaitForChannels")
	ErrFutureIsNotReady = errors.New("future is not ready after await")
	ErrFutureTimeout    = errors.New("timed out waiting for futures")
)

type WorkflowRunner any

type ActivityRunner any

type ActivityTypedRunner[IN any, OUT any] func(ctx context.Context, data IN) (OUT, error)

func WaitForFutures(ctx workflow.Context, timeout time.Duration, awaitables ...Awaitable) error {
	ok, err := workflow.AwaitWithTimeout(ctx, timeout, func() bool {
		for _, awaitable := range awaitables {
			if !awaitable.Future().IsReady() {
				return false
			}
		}

		return true
	})

	if !ok {
		var names []string

		for _, awaitable := range awaitables {
			names = append(names, awaitable.Name())
		}

		namesStr := strings.Join(names, "', '")

		return fmt.Errorf("%w: waiting for %d futures: '%s'", ErrFutureTimeout, len(awaitables), namesStr)
	}

	for _, awaitable := range awaitables {
		if !awaitable.Future().IsReady() {
			return fmt.Errorf("%w: '%s'", ErrFutureIsNotReady, awaitable.Name())
		}

		if err := awaitable.Future().Get(ctx, nil); err != nil {
			return fmt.Errorf("awaitable '%s' failed: %w", awaitable.Name(), err)
		}
	}

	return err
}

func WaitForChannels(
	ctx workflow.Context,
	channels ...workflow.Channel,
) error {
	channelCount := len(channels)
	if channelCount == 0 {
		return nil
	}

	received := make([]bool, channelCount)
	channelErrors := make([]error, channelCount)
	selector := workflow.NewSelector(ctx)

	for i, ch := range channels {
		idx := i

		selector.AddReceive(ch, func(c workflow.ReceiveChannel, more bool) {
			var err error
			c.Receive(ctx, &err)

			received[idx] = true
			channelErrors[idx] = err
		})
	}

	for done := 0; done < channelCount; {
		selector.Select(ctx)

		done = 0

		for _, ok := range received {
			if ok {
				done++
			}
		}
	}

	var channelsErrors []error

	for _, err := range channelErrors {
		if err != nil {
			channelsErrors = append(channelsErrors, err)
		}
	}

	if len(channelsErrors) > 0 {
		resultError := fmt.Errorf("%w", ErrWaitForChannels)

		for _, err := range channelsErrors {
			if err != nil {
				resultError = errors.Join(resultError, err)
			}
		}

		return resultError
	}

	return nil
}

//nolint:unparam
func ExecuteActivityAsync[IN any, OUT any](
	ctx workflow.Context,
	activityWrapper *ActivityTypedWrapper[IN, OUT],
	defaultValue OUT,
	data IN,
) (*ActivityFuture[OUT], error) {
	future := workflow.ExecuteActivity(getActivityCtx(ctx, activityWrapper), activityWrapper.activity.Name(), data)

	return NewActivityFuture[OUT](string(activityWrapper.activity.Name()), future, defaultValue), nil
}

func ExecuteActivitySync[IN any, OUT any](
	ctx workflow.Context,
	activityWrapper *ActivityTypedWrapper[IN, OUT],
	defaultValue OUT,
	data IN,
) (OUT, error) {
	var result OUT

	future, err := ExecuteActivityAsync(ctx, activityWrapper, defaultValue, data)
	if err != nil {
		return defaultValue, err
	}

	if err := future.Get(ctx, &result); err != nil {
		return defaultValue, err
	}

	return result, nil
}

func getActivityCtx[IN any, OUT any](ctx workflow.Context, activityWrapper *ActivityTypedWrapper[IN, OUT]) workflow.Context {
	activityOptions := workflow.ActivityOptions{
		TaskQueue:           string(activityWrapper.activity.Queue()),
		StartToCloseTimeout: activityWrapper.activity.Timeout(),
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts:    int32(activityWrapper.activity.MaxAttempts()), //nolint:gosec
			InitialInterval:    activityWrapper.activity.RetryInitialInterval(),
			BackoffCoefficient: activityWrapper.activity.RetryBackoffCoefficient(),
			MaximumInterval:    activityWrapper.activity.RetryMaximumInterval(),
		},
	}

	return workflow.WithActivityOptions(ctx, activityOptions)
}
