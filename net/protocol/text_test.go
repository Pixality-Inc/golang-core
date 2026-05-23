package protocol

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTextMarshalAddsLineDelimiter(t *testing.T) {
	t.Parallel()

	data, err := NewText().Marshal(t.Context(), "first", "second\n", "")
	require.NoError(t, err)
	require.Equal(t, []byte("first\nsecond\n\n"), data)
}

func TestTextMarshalReturnsContextErrorWhenCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := NewText().Marshal(ctx, "message")
	require.ErrorIs(t, err, context.Canceled)
}

func TestTextReadSplitsByNewLine(t *testing.T) {
	t.Parallel()

	messages, err := NewText().Read(t.Context(), strings.NewReader("first\nsecond\r\n\nlast"))
	require.NoError(t, err)
	require.Equal(t, []string{"first", "second", "", "last"}, readAll(messages))
}

func TestTextReadHandlesLinesAcrossChunks(t *testing.T) {
	t.Parallel()

	protocol := NewText()
	protocol.bufferSize = 2

	messages, err := protocol.Read(t.Context(), strings.NewReader("first\nsecond\n"))
	require.NoError(t, err)
	require.Equal(t, []string{"first", "second"}, readAll(messages))
}

func TestTextReadStopsWhenContextCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	reader, writer := io.Pipe()
	defer func() {
		_ = writer.Close()
	}()

	messages, err := NewText().Read(ctx, reader)
	require.NoError(t, err)

	cancel()
	requireClosed(t, messages)
}

func readAll[T any](ch <-chan T) []T {
	var result []T
	for data := range ch {
		result = append(result, data)
	}

	return result
}
