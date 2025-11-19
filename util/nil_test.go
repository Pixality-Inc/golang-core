package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsNil(t *testing.T) {
	t.Parallel()

	t.Run("nil value", func(t *testing.T) {
		t.Parallel()

		got := IsNil(nil)
		require.True(t, got)
	})

	t.Run("nil pointer", func(t *testing.T) {
		t.Parallel()

		var ptr *int

		got := IsNil(ptr)
		require.True(t, got)
	})

	t.Run("non-nil pointer", func(t *testing.T) {
		t.Parallel()

		val := 42
		ptr := &val
		got := IsNil(ptr)
		require.False(t, got)
	})

	t.Run("nil slice", func(t *testing.T) {
		t.Parallel()

		var slice []int

		got := IsNil(slice)
		require.True(t, got)
	})

	t.Run("empty slice", func(t *testing.T) {
		t.Parallel()

		slice := []int{}
		got := IsNil(slice)
		require.False(t, got)
	})

	t.Run("non-empty slice", func(t *testing.T) {
		t.Parallel()

		slice := []int{1, 2, 3}
		got := IsNil(slice)
		require.False(t, got)
	})

	t.Run("nil map", func(t *testing.T) {
		t.Parallel()

		var m map[string]int

		got := IsNil(m)
		require.True(t, got)
	})

	t.Run("empty map", func(t *testing.T) {
		t.Parallel()

		m := make(map[string]int)
		got := IsNil(m)
		require.False(t, got)
	})

	t.Run("non-empty map", func(t *testing.T) {
		t.Parallel()

		m := map[string]int{"key": 42}
		got := IsNil(m)
		require.False(t, got)
	})

	t.Run("nil channel", func(t *testing.T) {
		t.Parallel()

		var ch chan int

		got := IsNil(ch)
		require.True(t, got)
	})

	t.Run("non-nil channel", func(t *testing.T) {
		t.Parallel()

		ch := make(chan int)
		defer close(ch)

		got := IsNil(ch)
		require.False(t, got)
	})

	t.Run("nil function", func(t *testing.T) {
		t.Parallel()

		var fn func()

		got := IsNil(fn)
		require.True(t, got)
	})

	t.Run("non-nil function", func(t *testing.T) {
		t.Parallel()

		fn := func() {}
		got := IsNil(fn)
		require.False(t, got)
	})

	t.Run("nil interface", func(t *testing.T) {
		t.Parallel()

		var i any

		got := IsNil(i)
		require.True(t, got)
	})

	t.Run("interface with nil pointer", func(t *testing.T) {
		t.Parallel()

		var (
			ptr *int
			i   any = ptr
		)

		got := IsNil(i)
		require.True(t, got)
	})

	t.Run("interface with non-nil value", func(t *testing.T) {
		t.Parallel()

		var i any = 42

		got := IsNil(i)
		require.False(t, got)
	})

	t.Run("interface with non-nil pointer", func(t *testing.T) {
		t.Parallel()

		val := 42

		var i any = &val

		got := IsNil(i)
		require.False(t, got)
	})

	t.Run("non-nillable types - int", func(t *testing.T) {
		t.Parallel()

		got := IsNil(42)
		require.False(t, got)
	})

	t.Run("non-nillable types - string", func(t *testing.T) {
		t.Parallel()

		got := IsNil("hello")
		require.False(t, got)
	})

	t.Run("non-nillable types - empty string", func(t *testing.T) {
		t.Parallel()

		got := IsNil("")
		require.False(t, got)
	})

	t.Run("non-nillable types - bool", func(t *testing.T) {
		t.Parallel()

		got := IsNil(false)
		require.False(t, got)
	})

	t.Run("non-nillable types - struct", func(t *testing.T) {
		t.Parallel()

		type TestStruct struct {
			Field int
		}

		got := IsNil(TestStruct{})
		require.False(t, got)
	})

	t.Run("non-nillable types - zero value", func(t *testing.T) {
		t.Parallel()

		got := IsNil(0)
		require.False(t, got)
	})
}
