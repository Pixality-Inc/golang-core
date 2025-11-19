package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyIfNotNil(t *testing.T) {
	t.Parallel()

	value1 := "Hello"

	result1 := ApplyIfNotNil(&value1, func(value *string) *string {
		return MakeRef("[" + *value + "]")
	})
	require.Equal(t, MakeRef(`[Hello]`), result1)
	require.Equal(t, `[Hello]`, *result1)

	result2 := ApplyIfNotNil[string, string](nil, func(value *string) *string {
		return MakeRef("[" + *value + "]")
	})
	require.Nil(t, result2)
}

func TestApplyIfNotNilDefault(t *testing.T) {
	t.Parallel()

	value1 := "Hello"
	defaultValue := "__default__"

	result1 := ApplyIfNotNilDefault(&value1, defaultValue, func(value string) string {
		return "[" + value + "]"
	})
	require.Equal(t, `[Hello]`, result1)

	result2 := ApplyIfNotNilDefault(nil, defaultValue, func(value string) string {
		return "[" + value + "]"
	})
	require.Equal(t, defaultValue, result2)
}
