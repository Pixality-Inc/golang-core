package kafka

import "context"

type Pingable interface {
	IsConnected() bool
	Ping(ctx context.Context) error
}

type Lifetime interface {
	Pingable
	Stop() error
}
