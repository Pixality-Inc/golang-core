package net

import (
	"context"
)

type Connection interface {
	Id() ConnectionId
	Address() Address
	Write(ctx context.Context, message Message) error
	Close() error
}
