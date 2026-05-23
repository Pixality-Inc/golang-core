package net

import (
	"context"
)

type Connection[OUT any] interface {
	Id() ConnectionId
	Address() Addresses
	Write(ctx context.Context, message OUT) error
	Close() error
}
