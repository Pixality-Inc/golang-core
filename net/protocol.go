package net

import (
	"errors"
	"io"
)

const DefaultByteProtocolBufferSize = 32 * 1024

var errInvalidByteProtocolBufferSize = errors.New("invalid byte protocol buffer size")

type Protocol[INP, OUT any] interface {
	Marshal(data OUT) ([]byte, error)
	Read(reader io.Reader) (chan INP, error)
}
