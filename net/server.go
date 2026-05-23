package net

import "context"

type Server[INP, OUT any] interface {
	Start(ctx context.Context) error
	Stop() error
}
