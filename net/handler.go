package net

import "context"

type Handler[T any] interface {
	Handle(ctx context.Context, connection Connection[T]) (Client[T], error)
	Close() error
}
