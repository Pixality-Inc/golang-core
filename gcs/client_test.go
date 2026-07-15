package gcs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestWithUploadRetry verifies the opt-in contract: uploads retry only when the
// caller passes WithUploadRetry, so existing callers keep the SDK's default
// upload behavior.
func TestWithUploadRetry(t *testing.T) {
	t.Parallel()

	t.Run("off by default", func(t *testing.T) {
		t.Parallel()

		impl, ok := NewClient("cred", "name", "bucket", "base", "https://public").(*Impl)
		require.True(t, ok)
		require.False(t, impl.uploadRetry)
	})

	t.Run("enabled via option", func(t *testing.T) {
		t.Parallel()

		impl, ok := NewClient("cred", "name", "bucket", "base", "https://public", WithUploadRetry()).(*Impl)
		require.True(t, ok)
		require.True(t, impl.uploadRetry)
	})
}
