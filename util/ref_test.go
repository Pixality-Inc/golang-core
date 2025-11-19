package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMakeRef(t *testing.T) {
	t.Parallel()

	t.Run("int value", func(t *testing.T) {
		t.Parallel()

		value := 42
		got := MakeRef(value)

		require.NotNil(t, got)
		require.Equal(t, value, *got)
		require.Equal(t, &value, got) // Should be different addresses
	})

	t.Run("string value", func(t *testing.T) {
		t.Parallel()

		value := "hello"
		got := MakeRef(value)

		require.NotNil(t, got)
		require.Equal(t, value, *got)
	})

	t.Run("bool value", func(t *testing.T) {
		t.Parallel()

		value := true
		got := MakeRef(value)

		require.NotNil(t, got)
		require.Equal(t, value, *got)
	})

	t.Run("float value", func(t *testing.T) {
		t.Parallel()

		value := 3.14
		got := MakeRef(value)

		require.NotNil(t, got)
		require.InEpsilon(t, value, *got, 1e-9)
	})

	t.Run("zero int", func(t *testing.T) {
		t.Parallel()

		value := 0
		got := MakeRef(value)

		require.NotNil(t, got)
		require.Equal(t, value, *got)
	})

	t.Run("empty string", func(t *testing.T) {
		t.Parallel()

		value := ""
		got := MakeRef(value)

		require.NotNil(t, got)
		require.Equal(t, value, *got)
	})

	t.Run("struct value", func(t *testing.T) {
		t.Parallel()

		type TestStruct struct {
			Name  string
			Value int
		}

		value := TestStruct{Name: "test", Value: 42}
		got := MakeRef(value)

		require.NotNil(t, got)
		require.Equal(t, value, *got)
		require.Equal(t, "test", got.Name)
		require.Equal(t, 42, got.Value)
	})

	t.Run("slice value", func(t *testing.T) {
		t.Parallel()

		value := []int{1, 2, 3}
		got := MakeRef(value)

		require.NotNil(t, got)
		require.Equal(t, value, *got)
	})

	t.Run("map value", func(t *testing.T) {
		t.Parallel()

		value := map[string]int{"key": 42}
		got := MakeRef(value)

		require.NotNil(t, got)
		require.Equal(t, value, *got)
	})

	t.Run("pointer value", func(t *testing.T) {
		t.Parallel()

		innerValue := 42
		value := &innerValue
		got := MakeRef(value)

		require.NotNil(t, got)
		require.Equal(t, value, *got)
		require.Equal(t, innerValue, **got)
	})

	t.Run("nil slice", func(t *testing.T) {
		t.Parallel()

		var value []int

		got := MakeRef(value)

		require.NotNil(t, got)
		require.Nil(t, *got)
	})

	t.Run("nil map", func(t *testing.T) {
		t.Parallel()

		var value map[string]int

		got := MakeRef(value)

		require.NotNil(t, got)
		require.Nil(t, *got)
	})

	t.Run("nil pointer", func(t *testing.T) {
		t.Parallel()

		var value *int

		got := MakeRef(value)

		require.NotNil(t, got)
		require.Nil(t, *got)
	})

	t.Run("modifying through pointer", func(t *testing.T) {
		t.Parallel()

		value := 10
		got := MakeRef(value)

		// Modify through pointer
		*got = 20

		// Original value should be unchanged
		require.Equal(t, 10, value)
		// Pointer should have new value
		require.Equal(t, 20, *got)
	})
}
