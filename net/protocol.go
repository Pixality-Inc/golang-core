package net

import (
	"errors"
	"fmt"
	"io"
)

const DefaultByteProtocolBufferSize = 32 * 1024

var errInvalidByteProtocolBufferSize = errors.New("invalid byte protocol buffer size")

type Protocol[T any] interface {
	Marshal(data T) ([]byte, error)
	Read(reader io.Reader) (chan T, error)
}

type ByteProtocol struct {
	bufferSize int
}

func NewByteProtocol() *ByteProtocol {
	return &ByteProtocol{
		bufferSize: DefaultByteProtocolBufferSize,
	}
}

func (p *ByteProtocol) Marshal(data []byte) ([]byte, error) {
	return data, nil
}

func (p *ByteProtocol) Read(reader io.Reader) (chan []byte, error) {
	if p.bufferSize <= 0 {
		return nil, fmt.Errorf("%w: %d", errInvalidByteProtocolBufferSize, p.bufferSize)
	}

	ch := make(chan []byte, 1)

	go func() {
		defer close(ch)

		buffer := make([]byte, p.bufferSize)

		for {
			num, err := reader.Read(buffer)
			if num > 0 {
				data := make([]byte, num)
				copy(data, buffer[:num])

				ch <- data
			}

			if err != nil {
				return
			}
		}
	}()

	return ch, nil
}
