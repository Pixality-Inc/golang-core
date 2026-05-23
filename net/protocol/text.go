package protocol

import (
	"bytes"
	"context"
	"io"
	"strings"
)

type Text struct {
	bufferSize int
}

func NewText() *Text {
	return &Text{
		bufferSize: DefaultBufferSize,
	}
}

func (p *Text) Marshal(ctx context.Context, data ...string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	result := make([]byte, 0)

	for _, dataEntry := range data {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		result = append(result, dataEntry...)
		if !strings.HasSuffix(dataEntry, "\n") {
			result = append(result, '\n')
		}
	}

	return result, nil
}

func (p *Text) Read(ctx context.Context, reader io.Reader) (chan string, error) {
	chunks, err := readChunks(ctx, reader, p.bufferSize)
	if err != nil {
		return nil, err
	}

	ch := make(chan string, 1)

	select {
	case <-ctx.Done():
		close(ch)

		return ch, nil
	default:
	}

	go func() {
		defer close(ch)

		var pending []byte
		for chunk := range chunks {
			pending = append(pending, chunk...)

			for {
				idx := bytes.IndexByte(pending, '\n')
				if idx < 0 {
					break
				}

				line := trimCarriageReturn(pending[:idx])
				if !send(ctx, ch, string(line)) {
					return
				}

				pending = pending[idx+1:]
			}
		}

		if len(pending) > 0 {
			line := trimCarriageReturn(pending)
			_ = send(ctx, ch, string(line))
		}
	}()

	return ch, nil
}

func trimCarriageReturn(data []byte) []byte {
	return bytes.TrimSuffix(data, []byte("\r"))
}
