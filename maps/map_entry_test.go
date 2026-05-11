package maps

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapEntry(t *testing.T) {
	t.Parallel()

	entry := NewMapEntry("key", 42)

	require.Equal(t, "key", entry.Key())
	require.Equal(t, 42, entry.Value())
}
