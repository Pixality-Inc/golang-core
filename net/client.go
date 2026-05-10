package net

import "context"

type Client interface {
	OnWrite(ctx context.Context, message Message) error
	OnClose() error
}
