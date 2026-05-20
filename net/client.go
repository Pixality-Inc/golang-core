package net

import "context"

type Client[T any] interface {
	OnConnect(ctx context.Context) error
	OnWrite(ctx context.Context, message T) error
	OnClose() error
}
