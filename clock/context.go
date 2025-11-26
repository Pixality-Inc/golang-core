package clock

import "context"

type ContextKeyType string

const ContextKey = ContextKeyType("clock")

func GetClock(ctx context.Context) Clock {
	contextClock, ok := ctx.Value(ContextKey).(Clock)
	if !ok {
		return Default
	}

	return contextClock
}

func WithClock(ctx context.Context, clock Clock) context.Context {
	return context.WithValue(ctx, ContextKey, clock)
}
