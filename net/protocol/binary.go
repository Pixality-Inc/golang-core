package protocol

import (
	"context"
	"io"
)

type Binary struct {
	bufferSize int
}

func NewBinary() *Binary {
	return &Binary{
		bufferSize: DefaultBufferSize,
	}
}

func (p *Binary) Marshal(ctx context.Context, data ...[]byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	result := make([]byte, 0)

	for _, dataEntry := range data {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		result = append(result, dataEntry...)
	}

	return result, nil
}

func (p *Binary) Read(ctx context.Context, reader io.Reader) (chan []byte, error) {
	return readChunks(ctx, reader, p.bufferSize)
}
