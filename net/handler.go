package net

import "context"

type Handler interface {
	Handle(ctx context.Context, connection Connection) (Client, error)
	Close() error
}
