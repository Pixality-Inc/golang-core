package net

import "context"

type Server[T any] interface {
	Start(ctx context.Context) error
	Stop() error
}
