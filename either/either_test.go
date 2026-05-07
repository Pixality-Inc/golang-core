package either

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

var errEitherTest = errors.New("either test error")

func TestLeft(t *testing.T) {
	t.Parallel()

	eith := Left[string, int]("failed")

	require.True(t, eith.IsLeft())
	require.False(t, eith.IsRight())
	require.Equal(t, "failed", eith.Left())

	right, left := eith.Value()
	require.Zero(t, right)
	require.Equal(t, "failed", left)
}

func TestRight(t *testing.T) {
	t.Parallel()

	eith := Right[string, int](42)

	require.False(t, eith.IsLeft())
	require.True(t, eith.IsRight())
	require.Equal(t, 42, eith.Right())

	right, left := eith.Value()
	require.Equal(t, 42, right)
	require.Empty(t, left)
}

func TestError(t *testing.T) {
	t.Parallel()

	eith := Error[int](errEitherTest)

	require.True(t, eith.IsLeft())
	require.False(t, eith.IsRight())

	value, err := eith.Value()
	require.Zero(t, value)
	require.ErrorIs(t, err, errEitherTest)
}

func TestRightError(t *testing.T) {
	t.Parallel()

	eith := RightError("ok")

	require.False(t, eith.IsLeft())
	require.True(t, eith.IsRight())

	value, err := eith.Value()
	require.Equal(t, "ok", value)
	require.NoError(t, err)
}
