package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTernary(t *testing.T) {
	t.Parallel()

	result1 := Ternary(true, "true", "false")
	require.Equal(t, "true", result1)

	result2 := Ternary(false, "true", "false")
	require.Equal(t, "false", result2)
}

func TestTernaryFunc(t *testing.T) {
	t.Parallel()

	result1 := TernaryFunc(true, func() string { return "true" }, func() string { return "false" })
	require.Equal(t, "true", result1)

	result2 := TernaryFunc(false, func() string { return "true" }, func() string { return "false" })
	require.Equal(t, "false", result2)
}
