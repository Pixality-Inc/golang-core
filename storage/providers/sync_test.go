package providers

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/pixality-inc/golang-core/storage"
	"github.com/stretchr/testify/require"
)

func newDualOsSyncStorage(t *testing.T) (*SyncImpl, string, string) {
	t.Helper()

	root1 := t.TempDir()
	root2 := t.TempDir()

	s1 := storage.NewStorage(NewOsProvider(root1), NewNoUrlProvider(""))
	s2 := storage.NewStorage(NewOsProvider(root2), NewNoUrlProvider(""))

	return NewSync(s1, s2), root1, root2
}

func requireSameFileOnBothRoots(t *testing.T, root1, root2, rel string, want []byte) {
	t.Helper()

	got1, err := os.ReadFile(filepath.Join(root1, rel))
	require.NoError(t, err)
	got2, err := os.ReadFile(filepath.Join(root2, rel))
	require.NoError(t, err)

	require.Equal(t, want, got1)
	require.Equal(t, want, got2)
}

func requireMissingOnBothRoots(t *testing.T, root1, root2, rel string) {
	t.Helper()

	_, err := os.Stat(filepath.Join(root1, rel))
	require.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(root2, rel))
	require.True(t, os.IsNotExist(err))
}

func TestSyncStorage_FileExists_falseWhenMissingOnBoth(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	syncStore, _, _ := newDualOsSyncStorage(t)

	exists, err := syncStore.FileExists(ctx, "missing.txt")
	require.NoError(t, err)
	require.False(t, exists)
}

func TestSyncStorage_FileExists_falseWhenOnlyOnOneBackend(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	syncStore, root1, root2 := newDualOsSyncStorage(t)

	require.NoError(t, os.WriteFile(filepath.Join(root1, "only-a.txt"), []byte("x"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root2, "only-b.txt"), []byte("y"), 0o600))

	exists, err := syncStore.FileExists(ctx, "only-a.txt")
	require.NoError(t, err)
	require.False(t, exists)

	exists, err = syncStore.FileExists(ctx, "only-b.txt")
	require.NoError(t, err)
	require.False(t, exists)
}

func TestSyncStorage_FileExists_trueWhenPresentOnBoth(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	syncStore, root1, root2 := newDualOsSyncStorage(t)

	require.NoError(t, os.WriteFile(filepath.Join(root1, "x.txt"), []byte("1"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root2, "x.txt"), []byte("1"), 0o600))

	exists, err := syncStore.FileExists(ctx, "x.txt")
	require.NoError(t, err)
	require.True(t, exists)
}

func TestSyncStorage_Write_ReadFile_roundTrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	syncStore, root1, root2 := newDualOsSyncStorage(t)

	want := []byte("hello world")
	require.NoError(t, syncStore.Write(ctx, "sub/file.bin", bytes.NewReader(want)))

	requireSameFileOnBothRoots(t, root1, root2, filepath.Join("sub", "file.bin"), want)

	rc, err := syncStore.ReadFile(ctx, "sub/file.bin")
	require.NoError(t, err)
	t.Cleanup(func() { _ = rc.Close() })

	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestSyncStorage_ReadFile_notFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	syncStore, _, _ := newDualOsSyncStorage(t)

	_, err := syncStore.ReadFile(ctx, "nope.txt")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrStorageFailed)
}

func TestSyncStorage_DeleteFile_removesFromBoth(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	syncStore, root1, root2 := newDualOsSyncStorage(t)

	rel := "to-delete.txt"
	require.NoError(t, os.WriteFile(filepath.Join(root1, rel), []byte("x"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root2, rel), []byte("x"), 0o600))

	require.NoError(t, syncStore.DeleteFile(ctx, rel))

	requireMissingOnBothRoots(t, root1, root2, rel)
}

func TestSyncStorage_DeleteFile_errorWhenMissingOnEither(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	syncStore, root1, root2 := newDualOsSyncStorage(t)

	rel := "half.txt"
	require.NoError(t, os.WriteFile(filepath.Join(root1, rel), []byte("x"), 0o600))

	err := syncStore.DeleteFile(ctx, rel)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrStorageFailed)

	_, statErr := os.Stat(filepath.Join(root1, rel))
	require.True(t, os.IsNotExist(statErr))
	_, statErr = os.Stat(filepath.Join(root2, rel))
	require.True(t, os.IsNotExist(statErr))
}

func TestSyncStorage_DeleteDir_recursive(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	syncStore, root1, root2 := newDualOsSyncStorage(t)

	for _, root := range []string{root1, root2} {
		require.NoError(t, os.MkdirAll(filepath.Join(root, "d", "nested"), 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(root, "d", "nested", "f.txt"), []byte("x"), 0o600))
	}

	require.NoError(t, syncStore.DeleteDir(ctx, "d"))

	_, err := os.Stat(filepath.Join(root1, "d"))
	require.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(root2, "d"))
	require.True(t, os.IsNotExist(err))
}

func TestSyncStorage_MkDir_and_ReadDir(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	syncStore, root1, root2 := newDualOsSyncStorage(t)

	require.NoError(t, syncStore.MkDir(ctx, "a/b"))

	for _, root := range []string{root1, root2} {
		require.NoError(t, os.WriteFile(filepath.Join(root, "a", "b", "one.txt"), []byte("1"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(root, "a", "b", "two.txt"), []byte("2"), 0o600))
	}

	entries, err := syncStore.ReadDir(ctx, "a/b")
	require.NoError(t, err)
	require.Len(t, entries, 2)

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}

	require.ElementsMatch(t, []string{"one.txt", "two.txt"}, names)
}

func TestSyncStorage_ReadDir_notFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	syncStore, _, _ := newDualOsSyncStorage(t)

	_, err := syncStore.ReadDir(ctx, "no-such-dir")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrStorageFailed)
}

func TestSyncStorage_ReadDir_mismatchBetweenRoots(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	syncStore, root1, root2 := newDualOsSyncStorage(t)

	require.NoError(t, syncStore.MkDir(ctx, "m/dir"))
	require.NoError(t, os.WriteFile(filepath.Join(root1, "m", "dir", "a.txt"), []byte("1"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root2, "m", "dir", "b.txt"), []byte("2"), 0o600))

	_, err := syncStore.ReadDir(ctx, "m/dir")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrStorageFailed)
}

func TestSyncStorage_ReadDir_mismatchFileVsDirSameName(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	syncStore, root1, root2 := newDualOsSyncStorage(t)

	require.NoError(t, syncStore.MkDir(ctx, "m/dir"))
	require.NoError(t, os.WriteFile(filepath.Join(root1, "m", "dir", "same"), []byte("x"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(root2, "m", "dir", "same"), 0o700))

	_, err := syncStore.ReadDir(ctx, "m/dir")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrStorageFailed)
}

func TestSyncStorage_Multipart_replicatesToBothRoots(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	syncStore, root1, root2 := newDualOsSyncStorage(t)

	upload, err := syncStore.CreateMultipartUpload(ctx, "merged.txt")
	require.NoError(t, err)
	require.NotEmpty(t, upload.Id())

	chunk1, err := syncStore.UploadMultipartChunk(ctx, "merged.txt", upload, 1, bytes.NewReader([]byte("aa")), 2)
	require.NoError(t, err)
	chunk2, err := syncStore.UploadMultipartChunk(ctx, "merged.txt", upload, 2, bytes.NewReader([]byte("bb")), 2)
	require.NoError(t, err)

	require.NoError(t, syncStore.CompleteMultipartUpload(ctx, "merged.txt", upload, []storage.MultipartChunk{
		chunk1, chunk2,
	}))

	requireSameFileOnBothRoots(t, root1, root2, "merged.txt", []byte("aabb"))
}

func TestSyncStorage_Multipart_abort_cleansBothRoots(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	syncStore, root1, root2 := newDualOsSyncStorage(t)

	upload, err := syncStore.CreateMultipartUpload(ctx, "out.txt")
	require.NoError(t, err)

	_, err = syncStore.UploadMultipartChunk(ctx, "out.txt", upload, 1, bytes.NewReader([]byte("x")), 1)
	require.NoError(t, err)

	require.NoError(t, syncStore.AbortMultipartUpload(ctx, "out.txt", upload))

	for _, root := range []string{root1, root2} {
		_, err = os.Stat(filepath.Join(root, "out.txt.parts"))
		require.True(t, os.IsNotExist(err))
	}
}

func TestSyncStorage_Close_nilError(t *testing.T) {
	t.Parallel()

	syncStore, _, _ := newDualOsSyncStorage(t)
	require.NoError(t, syncStore.Close())
}

func TestSyncStorage_FileExists_propagatesStatError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	fileRoot := filepath.Join(root, "file-as-root")
	require.NoError(t, os.WriteFile(fileRoot, []byte("x"), 0o600))

	s1 := storage.NewStorage(NewOsProvider(fileRoot), NewNoUrlProvider(""))
	s2 := storage.NewStorage(NewOsProvider(t.TempDir()), NewNoUrlProvider(""))
	syncStore := NewSync(s1, s2)

	_, err := syncStore.FileExists(ctx, "anything")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrStorageFailed)
}
