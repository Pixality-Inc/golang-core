package net

import (
	"context"
)

type Connection[OUT any] interface {
	Id() ConnectionId
	Address() Addresses
	Write(ctx context.Context, messages ...OUT) error
	Close() error
}
