package net

import (
	"context"
)

type Connection[T any] interface {
	Id() ConnectionId
	Address() Addresses
	Write(ctx context.Context, message T) error
	Close() error
}
