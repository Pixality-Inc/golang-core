package iterator

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

var errTestRead = errors.New("read failed")

func TestReaderIterator(t *testing.T) {
	t.Parallel()

	iterator := NewReaderIterator(bytes.NewReader([]byte("hello")))

	values, err := Materialize(iterator)
	require.NoError(t, err)
	require.Equal(t, []byte("hello"), values)
}

func TestReaderIteratorHasNextDoesNotAdvance(t *testing.T) {
	t.Parallel()

	iterator := NewReaderIterator(bytes.NewReader([]byte{0, 1, 2}))

	require.True(t, iterator.HasNext())
	require.True(t, iterator.HasNext())

	for _, expected := range []byte{0, 1, 2} {
		require.Equal(t, expected, iterator.Next())
	}

	require.False(t, iterator.HasNext())
}

func TestReaderIteratorErr(t *testing.T) {
	t.Parallel()

	iterator := NewReaderIterator(errorReader{err: errTestRead})

	values, err := Materialize(iterator)
	require.ErrorIs(t, err, errTestRead)
	require.Empty(t, values)
}

type errorReader struct {
	err error
}

func (r errorReader) Read([]byte) (int, error) {
	return 0, r.err
}
