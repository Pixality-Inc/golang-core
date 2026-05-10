package iterator

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPeekableConsumable(t *testing.T) {
	t.Parallel()

	iterator := NewPeekableConsumable(NewReaderIterator(bytes.NewReader([]byte("abcd"))))

	value, err := iterator.Peek()
	require.NoError(t, err)
	require.Equal(t, byte('a'), value)

	values, err := iterator.PeekN(3)
	require.NoError(t, err)
	require.Equal(t, []byte("abc"), values)

	value, err = iterator.Peek()
	require.NoError(t, err)
	require.Equal(t, byte('a'), value)

	require.NoError(t, iterator.Consume2())

	first, second, err := iterator.Peek2()
	require.NoError(t, err)
	require.Equal(t, byte('c'), first)
	require.Equal(t, byte('d'), second)

	require.NoError(t, iterator.Consume2())

	_, err = iterator.Peek()
	require.ErrorIs(t, err, ErrNotEnoughItems)
}

func TestPeekableConsumableConsumeNDoesNotConsumeOnShortInput(t *testing.T) {
	t.Parallel()

	iterator := NewPeekableConsumable(NewReaderIterator(bytes.NewReader([]byte("ab"))))

	require.ErrorIs(t, iterator.ConsumeN(3), ErrNotEnoughItems)

	first, second, err := iterator.Peek2()
	require.NoError(t, err)
	require.Equal(t, byte('a'), first)
	require.Equal(t, byte('b'), second)

	require.NoError(t, iterator.Consume2())

	_, err = iterator.Peek()
	require.ErrorIs(t, err, ErrNotEnoughItems)
}

func TestPeekableConsumableNegativeCount(t *testing.T) {
	t.Parallel()

	iterator := NewPeekableConsumable(NewReaderIterator(bytes.NewReader([]byte("ab"))))

	_, err := iterator.PeekN(-1)
	require.ErrorIs(t, err, ErrNegativeCount)

	require.ErrorIs(t, iterator.ConsumeN(-1), ErrNegativeCount)
}

func TestPeekableConsumableReaderError(t *testing.T) {
	t.Parallel()

	iterator := NewPeekableConsumable(NewReaderIterator(errorReader{err: errTestRead}))

	_, err := iterator.Peek()
	require.ErrorIs(t, err, errTestRead)
}
