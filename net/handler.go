package net

import "context"

type Handler[INP, OUT any] interface {
	Handle(ctx context.Context, connection Connection[OUT]) (Client[INP], error)
	Close() error
}
