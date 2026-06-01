package storage_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pixality-inc/golang-core/storage"
	"github.com/pixality-inc/golang-core/storage/providers"
	"github.com/stretchr/testify/require"
)

func newOsStorage(t *testing.T) (storage.Storage, string) {
	t.Helper()

	root := t.TempDir()

	return storage.NewStorage(providers.NewOsProvider(root), providers.NewNoUrlProvider("")), root
}

// TestCopy_SameStorage_nativeCopy verifies the dst==src shortcut performs a real
// server-side copy via the provider (duplicates bytes, keeps the source).
func TestCopy_SameStorage_nativeCopy(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, root := newOsStorage(t)

	want := []byte("same-storage")
	require.NoError(t, store.Write(ctx, "a/src.bin", bytes.NewReader(want)))

	require.NoError(t, storage.Copy(ctx, store, "b/dst.bin", store, "a/src.bin"))

	got, err := os.ReadFile(filepath.Join(root, "b/dst.bin"))
	require.NoError(t, err)
	require.Equal(t, want, got)

	exists, err := store.FileExists(ctx, "a/src.bin")
	require.NoError(t, err)
	require.True(t, exists)
}

// TestCopy_DifferentStorages_streams verifies that across two distinct storages
// the helper falls back to streaming (ReadFile -> Write) and content matches.
func TestCopy_DifferentStorages_streams(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	src, _ := newOsStorage(t)
	dst, dstRoot := newOsStorage(t)

	want := []byte("cross-storage")
	require.NoError(t, src.Write(ctx, "src.bin", bytes.NewReader(want)))

	require.NoError(t, storage.Copy(ctx, dst, "dst.bin", src, "src.bin"))

	got, err := os.ReadFile(filepath.Join(dstRoot, "dst.bin"))
	require.NoError(t, err)
	require.Equal(t, want, got)
}

// TestMove_SameStorage_nativeMove verifies the dst==src shortcut moves via the provider.
func TestMove_SameStorage_nativeMove(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, root := newOsStorage(t)

	want := []byte("move-me")
	require.NoError(t, store.Write(ctx, "src.bin", bytes.NewReader(want)))

	require.NoError(t, storage.Move(ctx, store, "dst.bin", store, "src.bin"))

	got, err := os.ReadFile(filepath.Join(root, "dst.bin"))
	require.NoError(t, err)
	require.Equal(t, want, got)

	exists, err := store.FileExists(ctx, "src.bin")
	require.NoError(t, err)
	require.False(t, exists)
}
