package net

import "context"

type Client[INP any] interface {
	OnConnect(ctx context.Context) error
	OnWrite(ctx context.Context, message INP) error
	OnClose(ctx context.Context) error
}
