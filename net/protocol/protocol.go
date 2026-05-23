package protocol

import (
	"context"
	"io"
)

type Protocol[INP, OUT any] interface {
	Marshal(ctx context.Context, data ...OUT) ([]byte, error)
	Read(ctx context.Context, reader io.Reader) (chan INP, error)
}
